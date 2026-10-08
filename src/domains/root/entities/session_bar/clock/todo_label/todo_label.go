package todo_label

import (
	"errors"
	"fmt"
	"image/color"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/domains/root/core"
	"orivo/src/domains/root/entities/session_bar/clock/sessions"
	"orivo/src/domains/root/entities/session_bar/clock/todo_label/todo_picker"
	"orivo/src/systems/config"
	"orivo/src/systems/todoist"
	"orivo/src/systems/todos"
)

const statusSeconds = 4

type Timer interface {
	Accent() color.RGBA
	TodoID() string
	TodoText() string
	Stat(todoID string) sessions.Stat
	SetTodo(id, text string)
}

type syncResult struct {
	count int
	err   error
}

type TodoLabel struct {
	fonts  *core.Fonts
	timer  Timer
	picker *todo_picker.TodoPicker

	sync       func() (int, error)
	results    chan syncResult
	syncing    bool
	status     string
	statusLeft float32
	drawn      string
}

func New(fonts *core.Fonts, timer Timer) *TodoLabel {
	return &TodoLabel{
		fonts:   fonts,
		timer:   timer,
		picker:  todo_picker.New(fonts, timer),
		sync:    syncTodoist,
		results: make(chan syncResult, 1),
	}
}

func syncTodoist() (int, error) {
	all, err := todoist.Sync(config.TodoistAuth(), config.TodoistCache(), config.TodoistOutbox(), sessions.WorkedToday(config.Sessions()))
	if err != nil {
		return 0, err
	}
	if err := todoist.PushProgress(config.TodoistAuth(), config.TodoistOutbox()); err != nil {
		return 0, err
	}
	overdue, today := todos.Split(all, time.Now())
	return len(overdue) + len(today), nil
}

func (l *TodoLabel) Close() {
	l.picker.Close()
}

func (l *TodoLabel) DialogOpen() bool {
	return l.picker.IsOpen()
}

func (l *TodoLabel) Update(dt float32) {
	select {
	case result := <-l.results:
		switch {
		case errors.Is(result.err, todoist.ErrNotConnected):
			l.status = "Not connected to Todoist — run `orivo connect-todoist`"
		case errors.Is(result.err, todoist.ErrTokenRejected):
			l.status = "Todoist sign-in expired — run `orivo connect-todoist`"
		case errors.Is(result.err, todoist.ErrReadOnly):
			l.status = "Todoist sign-in is read-only — run `orivo connect-todoist`"
		case result.err != nil:
			l.status = "Sync failed: " + result.err.Error()
		case result.count == 1:
			l.status = "Synced 1 todo from Todoist"
		default:
			l.status = fmt.Sprintf("Synced %d todos from Todoist", result.count)
		}
		l.fonts.Need(l.status)
		l.syncing = false
		l.statusLeft = statusSeconds
	default:
		if !l.syncing && l.statusLeft > 0 {
			l.statusLeft -= dt
			if l.statusLeft <= 0 {
				l.status = ""
			}
		}
	}

	l.picker.Update(dt)
	if l.picker.IsOpen() {
		return
	}

	shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
	if shift && rl.IsKeyPressed(rl.KeyT) {
		l.timer.SetTodo("", "")
	}

	if !l.syncing && rl.IsKeyPressed(rl.KeyS) {
		l.syncing = true
		l.status = "Syncing Todoist…"
		go func() {
			count, err := l.sync()
			l.results <- syncResult{count: count, err: err}
		}()
	}
}

func (l *TodoLabel) Draw() {
	screen := core.CurrentScreen()
	s := screen.Scale
	timer := l.timer

	text := "No todo selected  [t] pick"
	if timer.TodoID() != "" {
		text = timer.TodoText()
		if stat := timer.Stat(timer.TodoID()); stat.Sessions > 0 {
			text += fmt.Sprintf("  ·  %d sessions  ·  %d min", stat.Sessions, stat.Secs/60)
		}
	}
	l.drawn = l.status
	if l.status != "" {
		text = l.status
	}

	body := l.fonts.Body
	body.DrawCentered(body.Fit(text, screen.Width-40*s), screen.Width/2, screen.Height-80*s, timer.Accent())

	l.picker.Draw()
}

// Changed reports whether a sync status came in or ran out since the last
// Draw.
func (l *TodoLabel) Changed() bool {
	return l.status != l.drawn
}
