package entities

import (
	"fmt"

	"github.com/mt-shihab26/orivo/src/core"
)

type TodoLabel struct {
	input *core.Input
	fonts *core.Fonts
	timer core.Timer
}

func NewTodoLabel(input *core.Input, fonts *core.Fonts, timer core.Timer) *TodoLabel {
	fonts.Need(timer.TodoText())
	return &TodoLabel{input: input, fonts: fonts, timer: timer}
}

func (l *TodoLabel) Close() {}

func (l *TodoLabel) Update(dt float32) {
	for _, key := range l.input.Keys {
		if key.Ch == 'T' {
			l.timer.SetTodo("", "")
		}
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
}
