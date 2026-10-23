package clock

import (
	"errors"
	"fmt"
	"image/color"
	"sync/atomic"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/domains/root/core"
	"orivo/src/domains/root/widgets/session_bar/clock/hints"
	"orivo/src/domains/root/widgets/session_bar/clock/ipc"
	"orivo/src/domains/root/widgets/session_bar/clock/notify"
	"orivo/src/domains/root/widgets/session_bar/clock/phase"
	"orivo/src/domains/root/widgets/session_bar/clock/reduce_dialog"
	"orivo/src/domains/root/widgets/session_bar/clock/sessions"
	"orivo/src/domains/root/widgets/session_bar/clock/store"
	"orivo/src/domains/root/widgets/session_bar/clock/todo_label"
	"orivo/src/systems/config"
	"orivo/src/systems/logx"
	"orivo/src/systems/todoist"
	"orivo/src/systems/todos"
)

const (
	saveEvery = 60
	flushGap  = time.Minute
)

type Clock struct {
	cfg      config.Timer
	fonts    *core.Fonts
	store    *store.Store
	history  *sessions.History
	server   *ipc.Server
	now      func() time.Time
	notify   func(summary, body string)
	progress func(todoID, line string)
	complete func(todoID string)
	flush    func() bool

	todo   *todo_label.TodoLabel
	reduce *reduce_dialog.ReduceDialog
	hints  *hints.Hints

	phase          phase.Phase
	running        bool
	showMillis     bool
	todoID         string
	todoText       string
	remaining      time.Duration
	startedAt      time.Time
	phaseStartedAt time.Time
	sinceSave      float32
	flushWanted    bool
	lastFlush      time.Time
	drawnText      string
}

func New(cfg config.Timer, fonts *core.Fonts) *Clock {
	c := newClock(cfg, fonts, config.Store(), config.Sessions(), config.TodoistCache())
	c.notify = notify.Send
	c.progress = queueProgress
	c.complete = queueDone
	c.flush = flushProgress
	return c
}

func newClock(cfg config.Timer, fonts *core.Fonts, storePath, sessionsPath, cachePath string) *Clock {
	history, err := sessions.Open(sessionsPath)
	if err != nil {
		logx.Error("failed to read %s: %v", sessionsPath, err)
	}

	c := &Clock{
		cfg:        cfg,
		fonts:      fonts,
		store:      store.Load(storePath),
		history:    history,
		now:        time.Now,
		showMillis: cfg.ShowMillis,
	}
	c.phase = phase.Named(c.store.Phase())
	c.todoID, c.todoText = c.store.Todo()
	c.restore()

	c.todo = todo_label.New(fonts, c, cachePath)
	c.reduce = reduce_dialog.New(fonts, c)
	c.hints = hints.New(fonts, c)
	return c
}

func (c *Clock) Attach() {
	c.fonts.Need(c.todoText)
	c.server = ipc.Serve(config.Socket())
	c.todo.Attach()
	c.reduce.Attach()
	c.hints.Attach()
	c.publish()
}

func (c *Clock) Detach() {
	c.hints.Detach()
	c.reduce.Detach()
	c.todo.Detach()
	c.server.Close()
	c.save()
}

func (c *Clock) Update(dt float32) {
	now := c.now()

	if !c.todo.DialogOpen() {
		c.reduce.Update(dt)
	}
	if !c.reduce.IsOpen() {
		c.todo.Update(dt)
	}
	c.hints.Update(dt)

	ctrl := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
	listening := !c.reduce.IsOpen() && !c.todo.DialogOpen() && !ctrl

	if listening {
		for _, k := range []struct {
			key    int32
			action func()
		}{
			{rl.KeySpace, c.Toggle},
			{rl.KeyR, c.Reset},
			{rl.KeyU, c.Undo},
			{rl.KeyN, c.Skip},
			{rl.KeyC, c.Done},
			{rl.KeyM, c.ToggleMillis},
		} {
			if rl.IsKeyPressed(k.key) {
				k.action()
			}
		}
	}

	if c.running && c.Remaining() == 0 {
		c.end(now, c.total(), false)
	}

	c.sinceSave += dt
	if c.sinceSave >= saveEvery {
		c.sinceSave = 0
		c.save()
	}

	// Send at once, but never sooner than flushGap after the last send, so
	// sessions ending close together do not hammer Todoist.
	if c.flushWanted && c.flush != nil && (c.lastFlush.IsZero() || now.Sub(c.lastFlush) >= flushGap) {
		if c.flush() {
			gap := "first sync"
			if !c.lastFlush.IsZero() {
				gap = now.Sub(c.lastFlush).Round(time.Second).String() + " since the last sync"
			}
			logSync("todoist: syncing session progress in the background (%s)", gap)
			c.flushWanted = false
			c.lastFlush = now
		}
	}

	c.publish()
}

func (c *Clock) Draw() {
	screen := core.CurrentScreen()
	s := screen.Scale
	fonts := c.fonts
	accent := c.Accent()
	remaining := c.Remaining()

	// Above the todo line and the hint buttons.
	top, bottom := 92*s, screen.Height-(hints.Height+34)*s
	radius := max(min((bottom-top)/2-6*s, screen.Width*0.38), 40*s)
	cx, cy := screen.Width/2, (top+bottom)/2
	center := rl.Vector2{X: cx, Y: cy}

	// Clicking the clock starts or pauses it, as Space does.
	if core.Hotspot(rl.Rectangle{X: cx - radius, Y: cy - radius, Width: 2 * radius, Height: 2 * radius}) {
		c.Toggle()
		remaining = c.Remaining()
	}

	thickness := max(7*s, 3)
	rl.DrawRing(center, radius-thickness, radius, 0, 360, 96, core.ColorTrack)
	elapsed := 1 - float32(remaining)/float32(c.total())
	if elapsed > 0 {
		rl.DrawRing(center, radius-thickness, radius, -90, -90+360*min(elapsed, 1), 96, accent)
	}

	text := clockText(remaining, c.showMillis)
	c.drawnText = text
	inner := (radius - thickness) * 2 * 0.76
	fonts.EnsureClock(min(inner/(0.6*float32(len(text))), radius*0.62))

	digits := fonts.Clock
	if width := digits.Width(text); width > inner {
		digits.Size *= inner / width
	}
	digits.DrawCentered(text, cx, cy-digits.Size/2, accent)

	fonts.Body.DrawCentered(c.phase.Label(), cx, cy-digits.Size/2-fonts.Body.Size-10*s, accent)

	status, statusColor := "Paused", core.ColorDim
	if c.running {
		status, statusColor = "Running", accent
	}
	fonts.Body.DrawCentered(status, cx, cy+digits.Size/2+10*s, statusColor)

	c.hints.Draw()
	// The todo line draws the picker after it, and the dialogs come last.
	c.todo.Draw()
	c.reduce.Draw()
}

func (c *Clock) Accent() color.RGBA {
	return c.phase.Color()
}

func (c *Clock) Remaining() time.Duration {
	if c.startedAt.IsZero() {
		return c.remaining
	}
	return max(c.remaining-c.now().Sub(c.startedAt), 0)
}

func (c *Clock) SessionsToday() float64 {
	return c.history.CountOn(c.now())
}

func (c *Clock) DailyGoal() int {
	return c.cfg.Goal()
}

func (c *Clock) TodoID() string {
	return c.todoID
}

func (c *Clock) TodoText() string {
	return c.todoText
}

func (c *Clock) Stat(todoID string) sessions.Stat {
	return c.history.StatOn(c.now(), todoID)
}

func (c *Clock) SetTodo(id, text string) {
	c.stash()

	c.todoID, c.todoText = id, text
	c.restore()

	if c.running {
		c.startedAt = c.now()
		if c.phaseStartedAt.IsZero() {
			c.phaseStartedAt = c.now()
			c.store.SetPhaseStartedAt(c.todoID, c.phaseStartedAt)
		}
	}

	c.store.SetTodo(c.todoID, c.todoText)
	c.store.Save()
}

func (c *Clock) Reduce(d time.Duration) {
	c.remaining = max(c.Remaining()-d, 0)
	if c.running {
		c.startedAt = c.now()
	}
	c.store.SetRemaining(c.todoID, c.remaining)
	c.store.Save()
}

// Toggle starts or pauses the timer.
func (c *Clock) Toggle() {
	now := c.now()
	if c.running {
		c.remaining = c.Remaining()
		c.startedAt = time.Time{}
		c.running = false
		c.store.SetRemaining(c.todoID, c.remaining)
		c.store.Save()
		return
	}
	if c.phaseStartedAt.IsZero() {
		c.phaseStartedAt = now
		c.store.SetPhaseStartedAt(c.todoID, now)
		c.store.Save()
	}
	c.startedAt = now
	c.running = true
}

// Reset puts the current phase back to its full time, paused.
func (c *Clock) Reset() {
	c.running = false
	c.rewind(c.now())
	c.store.Save()
}

// Skip ends the current phase as if its time had run out.
func (c *Clock) Skip() {
	c.end(c.now(), c.total(), false)
}

func (c *Clock) Done() {
	c.done(c.now())
}

func (c *Clock) Undo() {
	c.undo(c.now())
}

func (c *Clock) PickTodo() {
	c.todo.Pick()
}

func (c *Clock) ClearTodo() {
	c.SetTodo("", "")
}

func (c *Clock) OpenReduce() {
	c.reduce.Open()
}

func (c *Clock) Sync() {
	c.todo.Sync()
}

func (c *Clock) ToggleMillis() {
	c.showMillis = !c.showMillis
}

// done completes the selected todo, and ends a work session that has run at
// all, counting only the share of it that ran.
func (c *Clock) done(now time.Time) {
	finished := c.todoID != ""
	if ran := c.total() - c.Remaining(); c.phase == phase.Work && ran > 0 {
		c.end(now, ran, finished)
	} else if finished && c.notify != nil {
		c.notify("Todo Done", c.todoText)
	}
	if finished {
		c.finish()
	}
}

// end records the phase as a session that ran for ran, and moves on to the
// next phase. finished tells the notification the todo is done too.
func (c *Clock) end(now time.Time, ran time.Duration, finished bool) {
	if c.notify != nil {
		summary, body := c.phase.EndMessage()
		if finished {
			summary = "Todo Done"
		}
		if c.todoText != "" {
			body = c.todoText + " — " + body
		}
		c.notify(summary, body)
	}

	started := c.phaseStartedAt
	if started.IsZero() {
		started = now
	}
	session := sessions.Session{
		Phase:        c.phase.Key(),
		DurationSecs: int(ran.Seconds()),
		StartedAt:    started,
		EndedAt:      now,
		TodoID:       c.todoID,
		TodoText:     c.todoText,
	}
	if ran < c.total() {
		session.Share = ran.Seconds() / c.total().Seconds()
	}
	if err := c.history.Record(session); err != nil {
		logx.Error("failed to record session: %v", err)
	}
	if c.progress != nil && c.phase == phase.Work && c.todoID != "" {
		stat := c.history.StatOn(now, c.todoID)
		c.progress(c.todoID, todoist.ProgressTag(stat.Sessions, stat.Secs))
		c.flushWanted = true
	}

	switch {
	case c.phase != phase.Work:
		c.phase = phase.Work
		c.running = false
	// A session ended early still counts as one towards the long break.
	case c.history.EndedOn(now)%c.cfg.Interval() == 0:
		c.phase = phase.LongBreak
		c.running = true
	default:
		c.phase = phase.Break
		c.running = true
	}

	// Every todo's saved time belonged to the phase that just ended.
	c.store.ClearTimers()
	c.rewind(now)
	c.store.SetPhase(c.phase.Name())
	c.store.Save()
}

// undo takes back the last session recorded today and returns to its phase,
// paused at full time, as if it had never ended.
func (c *Clock) undo(now time.Time) {
	last, ok, err := c.history.Undo(now)
	if err != nil {
		logx.Error("failed to undo the last session: %v", err)
		return
	}
	if !ok {
		return
	}

	if c.progress != nil && last.Phase == phase.Work.Key() && last.TodoID != "" {
		// No tag at all once the todo has no session left today.
		tag := ""
		if stat := c.history.StatOn(now, last.TodoID); stat.Sessions > 0 {
			tag = todoist.ProgressTag(stat.Sessions, stat.Secs)
		}
		c.progress(last.TodoID, tag)
		c.flushWanted = true
	}

	c.phase = phase.Keyed(last.Phase)
	if c.notify != nil {
		body := "Back at its start, paused."
		if last.TodoText != "" {
			body = last.TodoText + " — " + body
		}
		c.notify(c.phase.Label()+" Undone", body)
	}
	c.running = false
	c.store.ClearTimers()
	c.rewind(now)
	c.store.SetPhase(c.phase.Name())
	c.store.Save()
}

// finish marks the selected todo done, to be completed in Todoist with the
// next send, and clears it.
func (c *Clock) finish() {
	if c.complete != nil {
		c.complete(c.todoID)
		c.flushWanted = true
	}
	c.SetTodo("", "")
}

func (c *Clock) total() time.Duration {
	return c.phase.Duration(c.cfg)
}

func (c *Clock) rewind(now time.Time) {
	c.remaining = c.total()
	c.startedAt, c.phaseStartedAt = time.Time{}, time.Time{}
	c.store.ClearRemaining(c.todoID)
	c.store.ClearPhaseStartedAt(c.todoID)
	if c.running {
		c.startedAt, c.phaseStartedAt = now, now
		c.store.SetPhaseStartedAt(c.todoID, now)
	}
}

func (c *Clock) stash() {
	c.store.SetRemaining(c.todoID, c.Remaining())
	if !c.phaseStartedAt.IsZero() {
		c.store.SetPhaseStartedAt(c.todoID, c.phaseStartedAt)
	}
}

func (c *Clock) restore() {
	remaining, ok := c.store.Remaining(c.todoID)
	if !ok {
		remaining = c.total()
	}
	c.remaining = remaining
	c.phaseStartedAt, _ = c.store.PhaseStartedAt(c.todoID)
}

func (c *Clock) save() {
	c.stash()
	c.store.Save()
}

func (c *Clock) publish() {
	status := ipc.Status{
		Phase:            c.phase.Key(),
		Label:            c.phase.Label(),
		IsRunning:        c.running,
		RemainingMillis:  c.Remaining().Milliseconds(),
		SessionsToday:    c.SessionsToday(),
		DailySessionGoal: c.DailyGoal(),
	}
	if c.todoID != "" {
		id, text := c.todoID, c.todoText
		status.TodoID = &id
		status.TodoText = &text
	}
	c.server.Publish(status)
}

// queueProgress only writes the line to the outbox; flushProgress sends it
// with whatever else is queued.
func queueProgress(todoID, line string) {
	if err := (todoist.Outbox{Path: config.TodoistOutbox()}).Add(todoID, line); err != nil {
		logx.Error("failed to queue Todoist progress: %v", err)
	}
}

// queueDone writes the todo to the completions flushProgress sends, and takes
// it out of the cache at once, so the picker no longer offers it.
func queueDone(todoID string) {
	if err := (todoist.Completions{Path: config.TodoistDone()}).Add(todoID); err != nil {
		logx.Error("failed to queue the Todoist completion: %v", err)
		return
	}
	if err := (todos.Cache{Path: config.TodoistCache()}).Remove(todoID); err != nil {
		logx.Error("failed to drop the done todo from the cache: %v", err)
	}
}

var flushing atomic.Bool

// logSync is swapped out in tests so they do not write to the real log.
var logSync = logx.Info

// flushProgress sends the outbox to Todoist in the background. It reports
// false when the last flush is still out, so the caller tries again later;
// what fails stays queued for the next session or sync.
func flushProgress() bool {
	if !flushing.CompareAndSwap(false, true) {
		return false
	}
	go func() {
		defer flushing.Store(false)
		err := todoist.PushProgress(config.TodoistAuth(), config.TodoistOutbox(), config.TodoistDone())
		if err != nil && !errors.Is(err, todoist.ErrNotConnected) {
			logx.Warn("failed to update Todoist progress: %v", err)
		}
	}()
	return true
}

func clockText(remaining time.Duration, showMillis bool) string {
	if showMillis {
		millis := remaining.Milliseconds()
		return fmt.Sprintf("%02d:%02d.%02d", millis/60000, millis/1000%60, millis%1000/10)
	}
	secs := int64((remaining + time.Second - 1) / time.Second)
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}

// Changed reports whether the clock looks different from its last Draw: a
// running timer once a second, or every hundredth while it shows milliseconds.
func (c *Clock) Changed() bool {
	return clockText(c.Remaining(), c.showMillis) != c.drawnText || c.todo.Changed()
}

// Next returns how long until the clock looks different on its own: the next
// second of a running timer, the next hundredth while it shows milliseconds
// (the clock shows hundredths, cut down rather than rounded), or
// 0 while it is paused.
func (c *Clock) Next() time.Duration {
	switch {
	case !c.running:
		return 0
	case c.showMillis:
		return c.Remaining()%(10*time.Millisecond) + time.Millisecond
	}
	// The shown seconds round up, so they change as a whole second passes.
	return c.Remaining()%time.Second + time.Millisecond
}
