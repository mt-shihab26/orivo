package clock

import (
	"fmt"
	"image/color"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/domains/root/core"
	"orivo/src/domains/root/entities/session_bar/clock/ipc"
	"orivo/src/domains/root/entities/session_bar/clock/notify"
	"orivo/src/domains/root/entities/session_bar/clock/phase"
	"orivo/src/domains/root/entities/session_bar/clock/reduce_dialog"
	"orivo/src/domains/root/entities/session_bar/clock/sessions"
	"orivo/src/domains/root/entities/session_bar/clock/store"
	"orivo/src/domains/root/entities/session_bar/clock/todo_label"
	"orivo/src/systems/config"
	"orivo/src/systems/logx"
)

const saveEvery = 60

type Clock struct {
	cfg     config.Timer
	fonts   *core.Fonts
	store   *store.Store
	history *sessions.History
	server  *ipc.Server
	now     func() time.Time
	notify  func(summary, body string)

	todo   *todo_label.TodoLabel
	reduce *reduce_dialog.ReduceDialog

	phase          phase.Phase
	running        bool
	showMillis     bool
	todoID         string
	todoText       string
	remaining      time.Duration
	startedAt      time.Time
	phaseStartedAt time.Time
	sinceSave      float32
}

func New(cfg config.Timer, fonts *core.Fonts) *Clock {
	c := newClock(cfg, fonts, config.Store(), config.Sessions())
	fonts.Need(c.todoText)
	c.server = ipc.Serve(config.Socket())
	notify.LoadSound()
	c.notify = notify.Send
	c.publish()
	return c
}

func newClock(cfg config.Timer, fonts *core.Fonts, storePath, sessionsPath string) *Clock {
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

	c.todo = todo_label.New(fonts, c)
	c.reduce = reduce_dialog.New(fonts, c)
	return c
}

func (c *Clock) Close() {
	c.reduce.Close()
	c.todo.Close()
	c.save()
	notify.UnloadSound()
}

func (c *Clock) Update(dt float32) {
	now := c.now()
	ended := false

	if !c.todo.DialogOpen() {
		c.reduce.Update(dt)
	}
	if !c.reduce.IsOpen() {
		c.todo.Update(dt)
	}

	ctrl := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
	listening := !c.reduce.IsOpen() && !c.todo.DialogOpen() && !ctrl

	if listening && rl.IsKeyPressed(rl.KeySpace) {
		if c.running {
			c.remaining = c.Remaining()
			c.startedAt = time.Time{}
			c.running = false
			c.store.SetRemaining(c.todoID, c.remaining)
			c.store.Save()
		} else {
			if c.phaseStartedAt.IsZero() {
				c.phaseStartedAt = now
				c.store.SetPhaseStartedAt(c.todoID, now)
				c.store.Save()
			}
			c.startedAt = now
			c.running = true
		}
	}

	if listening && rl.IsKeyPressed(rl.KeyR) {
		c.remaining = c.total()
		c.startedAt = time.Time{}
		c.phaseStartedAt = time.Time{}
		c.running = false
		c.store.ClearRemaining(c.todoID)
		c.store.ClearPhaseStartedAt(c.todoID)
		c.store.Save()
	}

	if listening && rl.IsKeyPressed(rl.KeyN) {
		ended = true
	}

	if listening && rl.IsKeyPressed(rl.KeyM) {
		c.showMillis = !c.showMillis
	}

	if c.running && c.Remaining() == 0 {
		ended = true
	}

	if ended {
		if c.notify != nil {
			summary, body := c.phase.EndMessage()
			if c.todoText != "" {
				body = c.todoText + " — " + body
			}
			c.notify(summary, body)
		}

		started := c.phaseStartedAt
		if started.IsZero() {
			started = now
		}
		err := c.history.Record(sessions.Session{
			Phase:        c.phase.Key(),
			DurationSecs: int(c.total().Seconds()),
			StartedAt:    started,
			EndedAt:      now,
			TodoID:       c.todoID,
			TodoText:     c.todoText,
		})
		if err != nil {
			logx.Error("failed to record session: %v", err)
		}

		switch {
		case c.phase != phase.Work:
			c.phase = phase.Work
			c.running = false
		case c.SessionsToday()%c.cfg.Interval() == 0:
			c.phase = phase.LongBreak
			c.running = true
		default:
			c.phase = phase.Break
			c.running = true
		}

		c.remaining = c.total()
		c.startedAt = time.Time{}
		c.phaseStartedAt = time.Time{}
		c.store.ClearPhaseStartedAt(c.todoID)
		if c.running {
			c.startedAt = now
			c.phaseStartedAt = now
			c.store.SetPhaseStartedAt(c.todoID, now)
		}

		c.store.ClearRemaining(c.todoID)
		c.store.SetPhase(c.phase.Name())
		c.store.Save()
	}

	c.sinceSave += dt
	if c.sinceSave >= saveEvery {
		c.sinceSave = 0
		c.save()
	}

	c.publish()
}

func (c *Clock) Draw() {
	screen := core.CurrentScreen()
	s := screen.Scale
	fonts := c.fonts
	accent := c.Accent()
	remaining := c.Remaining()

	top, bottom := 92*s, screen.Height-96*s
	radius := max(min((bottom-top)/2-6*s, screen.Width*0.38), 40*s)
	cx, cy := screen.Width/2, (top+bottom)/2
	center := rl.Vector2{X: cx, Y: cy}

	thickness := max(7*s, 3)
	rl.DrawRing(center, radius-thickness, radius, 0, 360, 96, core.ColorTrack)
	elapsed := 1 - float32(remaining)/float32(c.total())
	if elapsed > 0 {
		rl.DrawRing(center, radius-thickness, radius, -90, -90+360*min(elapsed, 1), 96, accent)
	}

	text := clockText(remaining, c.showMillis)
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

func (c *Clock) SessionsToday() int {
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
	return c.history.Stat(todoID)
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

func (c *Clock) total() time.Duration {
	return c.phase.Duration(c.cfg)
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

func clockText(remaining time.Duration, showMillis bool) string {
	if showMillis {
		millis := remaining.Milliseconds()
		return fmt.Sprintf("%02d:%02d.%02d", millis/60000, millis/1000%60, millis%1000/10)
	}
	secs := int64((remaining + time.Second - 1) / time.Second)
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}
