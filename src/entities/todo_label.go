package entities

import (
	"fmt"

	"github.com/mt-shihab26/orivo/src/core"
)

type TodoLabel struct {
	world *core.World
}

func NewTodoLabel(world *core.World) *TodoLabel {
	world.Fonts.Need(world.Timer.Snapshot().TodoText)
	return &TodoLabel{world: world}
}

func (l *TodoLabel) Close() {}

func (l *TodoLabel) Update(dt float32) {
	for _, key := range l.world.Keys {
		if key.Ch == 'T' {
			l.world.Timer.SetTodo("", "")
		}
	}
}

func (l *TodoLabel) Draw() {
	w := l.world
	snap := w.Snap

	text := "No todo selected  [t] pick"
	if snap.TodoID != "" {
		text = snap.TodoText
		if snap.Stat.Sessions > 0 {
			text += fmt.Sprintf("  ·  %d sessions  ·  %d min", snap.Stat.Sessions, snap.Stat.Secs/60)
		}
	}

	body := w.Fonts.Body
	body.DrawCentered(body.Fit(text, w.Width-40*w.Scale), w.Width/2, w.Height-80*w.Scale, w.Color)
}
