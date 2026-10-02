package entities

import (
	"image/color"
	"time"

	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/systems/ipc"
	"github.com/mt-shihab26/orivo/src/systems/logx"
	"github.com/mt-shihab26/orivo/src/systems/notify"
	"github.com/mt-shihab26/orivo/src/systems/paths"
	"github.com/mt-shihab26/orivo/src/systems/sessions"
	"github.com/mt-shihab26/orivo/src/systems/store"
)

const saveEvery = 60

type Phase int

const (
	Work Phase = iota
	Break
	LongBreak
)

func (p Phase) Label() string {
	switch p {
	case Break:
		return "Short Break"
	case LongBreak:
		return "Long Break"
	default:
		return "Work Session"
	}
}

func (p Phase) Key() string {
	switch p {
	case Break:
		return "break"
	case LongBreak:
		return "long_break"
	default:
		return "work"
	}
}

func (p Phase) Name() string {
	switch p {
	case Break:
		return "Break"
	case LongBreak:
		return "LongBreak"
	default:
		return "Work"
	}
}

func phaseNamed(name string) Phase {
	switch name {
	case "Break":
		return Break
	case "LongBreak":
		return LongBreak
	default:
		return Work
	}
}

func (p Phase) Duration(cfg config.Timer) time.Duration {
	switch p {
	case Break:
		return cfg.Break()
	case LongBreak:
		return cfg.LongBreak()
	default:
		return cfg.Work()
	}
}

func (p Phase) Color() color.RGBA {
	switch p {
	case Break:
		return color.RGBA{102, 187, 106, 255}
	case LongBreak:
		return color.RGBA{38, 198, 218, 255}
	default:
		return color.RGBA{239, 83, 80, 255}
	}
}

type Stat struct {
	Sessions int
	Secs     int
}

type Pomodoro struct {
	cfg          config.Timer
	input        *core.Input
	store        *store.Store
	sessionsPath string
	server       *ipc.Server
	now          func() time.Time

	Phase      Phase
	Running    bool
	ShowMillis bool
	TodoID     string
	TodoText   string
	OnPhaseEnd func(summary, body string)

	remaining      time.Duration
	startedAt      time.Time
	phaseStartedAt time.Time
	days           map[string]int
	stats          map[string]Stat
	sinceSave      float32
}

func NewPomodoro(cfg config.Timer, input *core.Input) *Pomodoro {
	p := newPomodoro(cfg, input, paths.Store(), paths.Sessions())
	p.server = ipc.Serve(paths.Socket())
	notify.LoadSound()
	p.OnPhaseEnd = notify.Send
	p.publish()
	return p
}

func newPomodoro(cfg config.Timer, input *core.Input, storePath, sessionsPath string) *Pomodoro {
	p := &Pomodoro{
		cfg:          cfg,
		input:        input,
		store:        store.Load(storePath),
		sessionsPath: sessionsPath,
		now:          time.Now,
		ShowMillis:   cfg.ShowMillis,
		days:         map[string]int{},
		stats:        map[string]Stat{},
	}

	history, err := sessions.Read(sessionsPath)
	if err != nil {
		logx.Error("failed to read %s: %v", sessionsPath, err)
	}
	for _, session := range history {
		p.count(session)
	}

	p.Phase = phaseNamed(p.store.Phase())
	p.TodoID, p.TodoText = p.store.Todo()
	p.restore()
	return p
}

func (p *Pomodoro) Close() {
	notify.UnloadSound()
	p.save()
}

func (p *Pomodoro) Update(dt float32) {
	for _, key := range p.input.Keys {
		switch key.Ch {
		case ' ':
			p.toggle()
		case 'r':
			p.reset()
		case 'n':
			p.advance()
		case 'm':
			p.ShowMillis = !p.ShowMillis
		}
	}

	if p.Running && p.Remaining() == 0 {
		p.advance()
	}

	p.sinceSave += dt
	if p.sinceSave >= saveEvery {
		p.sinceSave = 0
		p.save()
	}

	p.publish()
}

func (p *Pomodoro) Draw() {}

func (p *Pomodoro) Remaining() time.Duration {
	if p.startedAt.IsZero() {
		return p.remaining
	}
	return max(p.remaining-p.now().Sub(p.startedAt), 0)
}

func (p *Pomodoro) Total() time.Duration {
	return p.Phase.Duration(p.cfg)
}

func (p *Pomodoro) SessionsToday() int {
	return p.days[dayOf(p.now())]
}

func (p *Pomodoro) DailyGoal() int {
	return p.cfg.Goal()
}

func (p *Pomodoro) Stat(todoID string) Stat {
	return p.stats[todoID]
}

func (p *Pomodoro) SetTodo(id, text string) {
	p.stash()

	p.TodoID, p.TodoText = id, text
	p.restore()

	if p.Running {
		p.startedAt = p.now()
		if p.phaseStartedAt.IsZero() {
			p.phaseStartedAt = p.now()
			p.store.SetPhaseStartedAt(p.TodoID, p.phaseStartedAt)
		}
	}

	p.store.SetTodo(p.TodoID, p.TodoText)
	p.store.Save()
}

func (p *Pomodoro) Reduce(d time.Duration) {
	p.remaining = max(p.Remaining()-d, 0)
	if p.Running {
		p.startedAt = p.now()
	}
	p.store.SetRemaining(p.TodoID, p.remaining)
	p.store.Save()
}

func (p *Pomodoro) toggle() {
	if p.Running {
		p.remaining = p.Remaining()
		p.startedAt = time.Time{}
		p.Running = false
		p.store.SetRemaining(p.TodoID, p.remaining)
		p.store.Save()
		return
	}

	if p.phaseStartedAt.IsZero() {
		p.phaseStartedAt = p.now()
		p.store.SetPhaseStartedAt(p.TodoID, p.phaseStartedAt)
		p.store.Save()
	}
	p.startedAt = p.now()
	p.Running = true
}

func (p *Pomodoro) reset() {
	p.remaining = p.Total()
	p.startedAt = time.Time{}
	p.phaseStartedAt = time.Time{}
	p.Running = false
	p.store.ClearRemaining(p.TodoID)
	p.store.ClearPhaseStartedAt(p.TodoID)
	p.store.Save()
}

func (p *Pomodoro) advance() {
	p.announce()

	now := p.now()
	started := p.phaseStartedAt
	if started.IsZero() {
		started = now
	}
	p.store.ClearPhaseStartedAt(p.TodoID)

	session := sessions.Session{
		Phase:        p.Phase.Key(),
		DurationSecs: int(p.Total().Seconds()),
		StartedAt:    started,
		EndedAt:      now,
		TodoID:       p.TodoID,
		TodoText:     p.TodoText,
	}
	p.count(session)
	if err := sessions.Append(p.sessionsPath, session); err != nil {
		logx.Error("failed to record session: %v", err)
	}

	if p.Phase == Work {
		p.Phase = p.breakAfter(p.SessionsToday())
		p.Running = true
	} else {
		p.Phase = Work
		p.Running = false
	}

	p.remaining = p.Total()
	p.startedAt = time.Time{}
	p.phaseStartedAt = time.Time{}
	if p.Running {
		p.startedAt = now
		p.phaseStartedAt = now
		p.store.SetPhaseStartedAt(p.TodoID, now)
	}

	p.store.ClearRemaining(p.TodoID)
	p.store.SetPhase(p.Phase.Name())
	p.store.Save()
}

func (p *Pomodoro) breakAfter(sessionsToday int) Phase {
	if sessionsToday%p.cfg.Interval() == 0 {
		return LongBreak
	}
	return Break
}

func (p *Pomodoro) announce() {
	if p.OnPhaseEnd == nil {
		return
	}

	summary, body := "Work Session Complete", "Time for a break!"
	switch p.Phase {
	case Break:
		summary, body = "Break Complete", "Ready to focus?"
	case LongBreak:
		summary, body = "Long Break Complete", "Ready to focus?"
	}
	if p.TodoText != "" {
		body = p.TodoText + " — " + body
	}
	p.OnPhaseEnd(summary, body)
}

func (p *Pomodoro) count(session sessions.Session) {
	if session.Phase != Work.Key() {
		return
	}
	p.days[dayOf(session.EndedAt)]++
	if session.TodoID != "" {
		stat := p.stats[session.TodoID]
		stat.Sessions++
		stat.Secs += session.DurationSecs
		p.stats[session.TodoID] = stat
	}
}

func (p *Pomodoro) stash() {
	p.store.SetRemaining(p.TodoID, p.Remaining())
	if !p.phaseStartedAt.IsZero() {
		p.store.SetPhaseStartedAt(p.TodoID, p.phaseStartedAt)
	}
}

func (p *Pomodoro) restore() {
	remaining, ok := p.store.Remaining(p.TodoID)
	if !ok {
		remaining = p.Total()
	}
	p.remaining = remaining
	p.phaseStartedAt, _ = p.store.PhaseStartedAt(p.TodoID)
}

func (p *Pomodoro) save() {
	p.stash()
	p.store.Save()
}

func (p *Pomodoro) publish() {
	status := ipc.Status{
		Phase:            p.Phase.Key(),
		Label:            p.Phase.Label(),
		IsRunning:        p.Running,
		RemainingMillis:  p.Remaining().Milliseconds(),
		SessionsToday:    p.SessionsToday(),
		DailySessionGoal: p.DailyGoal(),
	}
	if p.TodoID != "" {
		id, text := p.TodoID, p.TodoText
		status.TodoID = &id
		status.TodoText = &text
	}
	p.server.Publish(status)
}

func dayOf(t time.Time) string {
	return t.Local().Format(time.DateOnly)
}
