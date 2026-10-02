package app

import (
	"image/color"
	"os"
	"os/signal"
	"slices"
	"sync/atomic"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/entities"
	"github.com/mt-shihab26/orivo/src/systems/ipc"
	"github.com/mt-shihab26/orivo/src/systems/logx"
	"github.com/mt-shihab26/orivo/src/systems/notify"
	"github.com/mt-shihab26/orivo/src/systems/paths"
	"github.com/mt-shihab26/orivo/src/systems/sessions"
	"github.com/mt-shihab26/orivo/src/systems/store"
)

const saveEvery = 60

type phase int

const (
	work phase = iota
	shortBreak
	longBreak
)

func (p phase) label() string {
	switch p {
	case shortBreak:
		return "Short Break"
	case longBreak:
		return "Long Break"
	default:
		return "Work Session"
	}
}

func (p phase) key() string {
	switch p {
	case shortBreak:
		return "break"
	case longBreak:
		return "long_break"
	default:
		return "work"
	}
}

func (p phase) name() string {
	switch p {
	case shortBreak:
		return "Break"
	case longBreak:
		return "LongBreak"
	default:
		return "Work"
	}
}

func phaseNamed(name string) phase {
	switch name {
	case "Break":
		return shortBreak
	case "LongBreak":
		return longBreak
	default:
		return work
	}
}

func (p phase) duration(cfg config.Timer) time.Duration {
	switch p {
	case shortBreak:
		return cfg.Break()
	case longBreak:
		return cfg.LongBreak()
	default:
		return cfg.Work()
	}
}

func (p phase) color() color.RGBA {
	switch p {
	case shortBreak:
		return color.RGBA{102, 187, 106, 255}
	case longBreak:
		return color.RGBA{38, 198, 218, 255}
	default:
		return color.RGBA{239, 83, 80, 255}
	}
}

type App struct {
	cfg      config.Timer
	fonts    *core.Fonts
	input    *core.Input
	entities []core.Entity
	quit     atomic.Bool

	store        *store.Store
	sessionsPath string
	server       *ipc.Server
	now          func() time.Time
	onPhaseEnd   func(summary, body string)

	phase          phase
	running        bool
	showMillis     bool
	todoID         string
	todoText       string
	remaining      time.Duration
	startedAt      time.Time
	phaseStartedAt time.Time
	days           map[string]int
	stats          map[string]core.Stat
	sinceSave      float32
}

func New(cfg config.Config) *App {
	rl.SetTraceLogLevel(rl.LogWarning)
	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagMsaa4xHint)
	rl.InitWindow(core.BaseWidth, core.BaseHeight, "Orivo")
	rl.SetWindowMinSize(420, 360)
	rl.SetTargetFPS(60)
	rl.SetExitKey(0)
	rl.InitAudioDevice()

	a := load(cfg.Timer, paths.Store(), paths.Sessions())
	a.fonts = core.NewFonts(cfg.Font)
	a.server = ipc.Serve(paths.Socket())
	notify.LoadSound()
	a.onPhaseEnd = notify.Send

	a.entities = []core.Entity{
		entities.NewTopBar(a.input, a.fonts, cfg.ShowFPS, a.Quit),
		entities.NewSessionBar(a.fonts, a),
		entities.NewClock(a.fonts, a),
		entities.NewTodoLabel(a.input, a.fonts, a),
		entities.NewHints(a.fonts),
		entities.NewTodoPicker(a.input, a.fonts, a),
		entities.NewReduceDialog(a.input, a.fonts, a),
	}

	a.publish()
	a.quitOnSignal()
	return a
}

func load(cfg config.Timer, storePath, sessionsPath string) *App {
	a := &App{
		cfg:          cfg,
		input:        &core.Input{},
		store:        store.Load(storePath),
		sessionsPath: sessionsPath,
		now:          time.Now,
		showMillis:   cfg.ShowMillis,
		days:         map[string]int{},
		stats:        map[string]core.Stat{},
	}

	history, err := sessions.Read(sessionsPath)
	if err != nil {
		logx.Error("failed to read %s: %v", sessionsPath, err)
	}
	for _, session := range history {
		a.count(session)
	}

	a.phase = phaseNamed(a.store.Phase())
	a.todoID, a.todoText = a.store.Todo()
	a.restore()
	return a
}

func (a *App) Close() {
	for _, entity := range slices.Backward(a.entities) {
		entity.Close()
	}
	a.save()
	notify.UnloadSound()
	a.fonts.Close()
	rl.CloseAudioDevice()
	rl.CloseWindow()
}

func (a *App) Quit() {
	a.quit.Store(true)
}

func (a *App) Run() {
	for !rl.WindowShouldClose() && !a.quit.Load() {
		a.Update(rl.GetFrameTime())
		a.Draw()
	}
}

func (a *App) Update(dt float32) {
	keys := core.ReadKeys()
	modal := a.openModal()

	if modal == nil {
		a.handleKeys(keys)
	}
	a.tick(dt)

	for _, entity := range a.entities {
		a.input.Keys = nil
		if modal == nil || entity == modal {
			a.input.Keys = keys
		}
		entity.Update(dt)
	}
}

func (a *App) Draw() {
	a.fonts.Ensure(core.CurrentScreen().Scale)

	rl.BeginDrawing()
	defer rl.EndDrawing()
	rl.ClearBackground(core.ColorBackground)

	for _, entity := range a.entities {
		entity.Draw()
	}
}

func (a *App) openModal() core.Entity {
	for _, entity := range a.entities {
		if modal, ok := entity.(core.Modal); ok && modal.IsOpen() {
			return entity
		}
	}
	return nil
}

func (a *App) quitOnSignal() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		a.Quit()
	}()
}

func (a *App) handleKeys(keys []core.Key) {
	for _, key := range keys {
		switch key.Ch {
		case ' ':
			a.toggle()
		case 'r':
			a.reset()
		case 'n':
			a.advance()
		case 'm':
			a.showMillis = !a.showMillis
		}
	}
}

func (a *App) tick(dt float32) {
	if a.running && a.Remaining() == 0 {
		a.advance()
	}

	a.sinceSave += dt
	if a.sinceSave >= saveEvery {
		a.sinceSave = 0
		a.save()
	}

	a.publish()
}

func (a *App) PhaseLabel() string {
	return a.phase.label()
}

func (a *App) Accent() color.RGBA {
	return a.phase.color()
}

func (a *App) Running() bool {
	return a.running
}

func (a *App) ShowMillis() bool {
	return a.showMillis
}

func (a *App) Remaining() time.Duration {
	if a.startedAt.IsZero() {
		return a.remaining
	}
	return max(a.remaining-a.now().Sub(a.startedAt), 0)
}

func (a *App) Total() time.Duration {
	return a.phase.duration(a.cfg)
}

func (a *App) SessionsToday() int {
	return a.days[dayOf(a.now())]
}

func (a *App) DailyGoal() int {
	return a.cfg.Goal()
}

func (a *App) TodoID() string {
	return a.todoID
}

func (a *App) TodoText() string {
	return a.todoText
}

func (a *App) Stat(todoID string) core.Stat {
	return a.stats[todoID]
}

func (a *App) SetTodo(id, text string) {
	a.stash()

	a.todoID, a.todoText = id, text
	a.restore()

	if a.running {
		a.startedAt = a.now()
		if a.phaseStartedAt.IsZero() {
			a.phaseStartedAt = a.now()
			a.store.SetPhaseStartedAt(a.todoID, a.phaseStartedAt)
		}
	}

	a.store.SetTodo(a.todoID, a.todoText)
	a.store.Save()
}

func (a *App) Reduce(d time.Duration) {
	a.remaining = max(a.Remaining()-d, 0)
	if a.running {
		a.startedAt = a.now()
	}
	a.store.SetRemaining(a.todoID, a.remaining)
	a.store.Save()
}

func (a *App) toggle() {
	if a.running {
		a.remaining = a.Remaining()
		a.startedAt = time.Time{}
		a.running = false
		a.store.SetRemaining(a.todoID, a.remaining)
		a.store.Save()
		return
	}

	if a.phaseStartedAt.IsZero() {
		a.phaseStartedAt = a.now()
		a.store.SetPhaseStartedAt(a.todoID, a.phaseStartedAt)
		a.store.Save()
	}
	a.startedAt = a.now()
	a.running = true
}

func (a *App) reset() {
	a.remaining = a.Total()
	a.startedAt = time.Time{}
	a.phaseStartedAt = time.Time{}
	a.running = false
	a.store.ClearRemaining(a.todoID)
	a.store.ClearPhaseStartedAt(a.todoID)
	a.store.Save()
}

func (a *App) advance() {
	a.announce()

	now := a.now()
	started := a.phaseStartedAt
	if started.IsZero() {
		started = now
	}
	a.store.ClearPhaseStartedAt(a.todoID)

	session := sessions.Session{
		Phase:        a.phase.key(),
		DurationSecs: int(a.Total().Seconds()),
		StartedAt:    started,
		EndedAt:      now,
		TodoID:       a.todoID,
		TodoText:     a.todoText,
	}
	a.count(session)
	if err := sessions.Append(a.sessionsPath, session); err != nil {
		logx.Error("failed to record session: %v", err)
	}

	if a.phase == work {
		a.phase = a.breakAfter(a.SessionsToday())
		a.running = true
	} else {
		a.phase = work
		a.running = false
	}

	a.remaining = a.Total()
	a.startedAt = time.Time{}
	a.phaseStartedAt = time.Time{}
	if a.running {
		a.startedAt = now
		a.phaseStartedAt = now
		a.store.SetPhaseStartedAt(a.todoID, now)
	}

	a.store.ClearRemaining(a.todoID)
	a.store.SetPhase(a.phase.name())
	a.store.Save()
}

func (a *App) breakAfter(sessionsToday int) phase {
	if sessionsToday%a.cfg.Interval() == 0 {
		return longBreak
	}
	return shortBreak
}

func (a *App) announce() {
	if a.onPhaseEnd == nil {
		return
	}

	summary, body := "Work Session Complete", "Time for a break!"
	switch a.phase {
	case shortBreak:
		summary, body = "Break Complete", "Ready to focus?"
	case longBreak:
		summary, body = "Long Break Complete", "Ready to focus?"
	}
	if a.todoText != "" {
		body = a.todoText + " — " + body
	}
	a.onPhaseEnd(summary, body)
}

func (a *App) count(session sessions.Session) {
	if session.Phase != work.key() {
		return
	}
	a.days[dayOf(session.EndedAt)]++
	if session.TodoID != "" {
		stat := a.stats[session.TodoID]
		stat.Sessions++
		stat.Secs += session.DurationSecs
		a.stats[session.TodoID] = stat
	}
}

func (a *App) stash() {
	a.store.SetRemaining(a.todoID, a.Remaining())
	if !a.phaseStartedAt.IsZero() {
		a.store.SetPhaseStartedAt(a.todoID, a.phaseStartedAt)
	}
}

func (a *App) restore() {
	remaining, ok := a.store.Remaining(a.todoID)
	if !ok {
		remaining = a.Total()
	}
	a.remaining = remaining
	a.phaseStartedAt, _ = a.store.PhaseStartedAt(a.todoID)
}

func (a *App) save() {
	a.stash()
	a.store.Save()
}

func (a *App) publish() {
	status := ipc.Status{
		Phase:            a.phase.key(),
		Label:            a.phase.label(),
		IsRunning:        a.running,
		RemainingMillis:  a.Remaining().Milliseconds(),
		SessionsToday:    a.SessionsToday(),
		DailySessionGoal: a.DailyGoal(),
	}
	if a.todoID != "" {
		id, text := a.todoID, a.todoText
		status.TodoID = &id
		status.TodoText = &text
	}
	a.server.Publish(status)
}

func dayOf(t time.Time) string {
	return t.Local().Format(time.DateOnly)
}
