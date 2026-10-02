package entities

import (
	"fmt"

	"github.com/mt-shihab26/orivo/src/core"
)

type TodoLabel struct {
	world    *core.World
	pomodoro *Pomodoro
}

func NewTodoLabel(world *core.World, pomodoro *Pomodoro) *TodoLabel {
	world.Fonts.Need(pomodoro.TodoText)
	return &TodoLabel{world: world, pomodoro: pomodoro}
}

func (l *TodoLabel) Close() {}

func (l *TodoLabel) Update(dt float32) {
	for _, key := range l.world.Keys {
		if key.Ch == 'T' {
			l.pomodoro.SetTodo("", "")
		}
	}
}

func (l *TodoLabel) Draw() {
	w := l.world
	pomodoro := l.pomodoro

	text := "No todo selected  [t] pick"
	if pomodoro.TodoID != "" {
		text = pomodoro.TodoText
		if stat := pomodoro.Stat(pomodoro.TodoID); stat.Sessions > 0 {
			text += fmt.Sprintf("  ·  %d sessions  ·  %d min", stat.Sessions, stat.Secs/60)
		}
	}

	body := w.Fonts.Body
	body.DrawCentered(body.Fit(text, w.Width-40*w.Scale), w.Width/2, w.Height-80*w.Scale, w.Color)
}
