package entities

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
)

type TodoLabel struct {
	dialogOpen *bool
	fonts      *core.Fonts
	clock      *Clock
}

func NewTodoLabel(dialogOpen *bool, fonts *core.Fonts, clock *Clock) *TodoLabel {
	fonts.Need(clock.TodoText())
	return &TodoLabel{dialogOpen: dialogOpen, fonts: fonts, clock: clock}
}

func (l *TodoLabel) Close() {}

func (l *TodoLabel) Update(dt float32) {
	shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

	if !*l.dialogOpen && shift && rl.IsKeyPressed(rl.KeyT) {
		l.clock.SetTodo("", "")
	}
}

func (l *TodoLabel) Draw() {
	screen := core.CurrentScreen()
	s := screen.Scale
	clock := l.clock

	text := "No todo selected  [t] pick"
	if clock.TodoID() != "" {
		text = clock.TodoText()
		if stat := clock.Stat(clock.TodoID()); stat.Sessions > 0 {
			text += fmt.Sprintf("  ·  %d sessions  ·  %d min", stat.Sessions, stat.Secs/60)
		}
	}

	body := l.fonts.Body
	body.DrawCentered(body.Fit(text, screen.Width-40*s), screen.Width/2, screen.Height-80*s, clock.Accent())
}
