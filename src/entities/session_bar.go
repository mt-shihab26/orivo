package entities

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
)

type SessionBar struct {
	world *core.World
}

func NewSessionBar(world *core.World) *SessionBar {
	return &SessionBar{world: world}
}

func (b *SessionBar) Close() {}

func (b *SessionBar) Update(dt float32) {}

func (b *SessionBar) Draw() {
	w := b.world
	s := w.Scale
	y := 42 * s

	label := fmt.Sprintf("Session %d / %d", w.Snap.SessionsToday, w.Snap.DailyGoal)
	w.Fonts.Body.DrawCentered(label, w.Width/2, y, w.Color)

	bar := rl.Rectangle{X: w.Width * 0.25, Y: y + 32*s, Width: w.Width * 0.5, Height: 6 * s}
	ratio := min(float32(w.Snap.SessionsToday)/float32(w.Snap.DailyGoal), 1)

	rl.DrawRectangleRounded(bar, 1, 6, core.ColorTrack)
	if ratio > 0 {
		bar.Width = max(bar.Width*ratio, bar.Height)
		rl.DrawRectangleRounded(bar, 1, 6, w.Color)
	}
}
