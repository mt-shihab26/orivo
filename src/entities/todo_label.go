package entities

import (
	"fmt"

	"github.com/mt-shihab26/orivo/src/core"
)

type TodoLabel struct {
	input    *core.Input
	fonts    *core.Fonts
	pomodoro *Pomodoro
}

func NewTodoLabel(input *core.Input, fonts *core.Fonts, pomodoro *Pomodoro) *TodoLabel {
	fonts.Need(pomodoro.TodoText)
	return &TodoLabel{input: input, fonts: fonts, pomodoro: pomodoro}
}

func (l *TodoLabel) Close() {}

func (l *TodoLabel) Update(dt float32) {
	for _, key := range l.input.Keys {
		if key.Ch == 'T' {
			l.pomodoro.SetTodo("", "")
		}
	}
}

func (l *TodoLabel) Draw() {
	screen := core.CurrentScreen()
	s := screen.Scale
	pomodoro := l.pomodoro

	text := "No todo selected  [t] pick"
	if pomodoro.TodoID != "" {
		text = pomodoro.TodoText
		if stat := pomodoro.Stat(pomodoro.TodoID); stat.Sessions > 0 {
			text += fmt.Sprintf("  ·  %d sessions  ·  %d min", stat.Sessions, stat.Secs/60)
		}
	}

	body := l.fonts.Body
	body.DrawCentered(body.Fit(text, screen.Width-40*s), screen.Width/2, screen.Height-80*s, pomodoro.Phase.Color())
}
