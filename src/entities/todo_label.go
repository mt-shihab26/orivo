package entities

import (
	"fmt"

	"github.com/mt-shihab26/orivo/src/core"
)

type TodoLabel struct {
	input *core.Input
	fonts *core.Fonts
	clock *Clock
}

func NewTodoLabel(input *core.Input, fonts *core.Fonts, clock *Clock) *TodoLabel {
	fonts.Need(clock.TodoText())
	return &TodoLabel{input: input, fonts: fonts, clock: clock}
}

func (l *TodoLabel) Close() {}

func (l *TodoLabel) Update(dt float32) {
	for _, key := range l.input.KeysFor(l) {
		if key.Ch == 'T' {
			l.clock.SetTodo("", "")
		}
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
