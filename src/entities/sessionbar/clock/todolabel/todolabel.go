package todolabel

import (
	"fmt"
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/entities/sessionbar/clock/todolabel/todopicker"
	"github.com/mt-shihab26/orivo/src/systems/sessions"
)

type Timer interface {
	Accent() color.RGBA
	TodoID() string
	TodoText() string
	Stat(todoID string) sessions.Stat
	SetTodo(id, text string)
}

type TodoLabel struct {
	fonts  *core.Fonts
	timer  Timer
	picker *todopicker.TodoPicker
}

func New(fonts *core.Fonts, timer Timer) *TodoLabel {
	return &TodoLabel{fonts: fonts, timer: timer, picker: todopicker.New(fonts, timer)}
}

func (l *TodoLabel) Close() {
	l.picker.Close()
}

func (l *TodoLabel) DialogOpen() bool {
	return l.picker.IsOpen()
}

func (l *TodoLabel) Update(dt float32) {
	l.picker.Update(dt)
	if l.picker.IsOpen() {
		return
	}

	shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
	if shift && rl.IsKeyPressed(rl.KeyT) {
		l.timer.SetTodo("", "")
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

	body := l.fonts.Body
	body.DrawCentered(body.Fit(text, screen.Width-40*s), screen.Width/2, screen.Height-80*s, timer.Accent())

	l.picker.Draw()
}
