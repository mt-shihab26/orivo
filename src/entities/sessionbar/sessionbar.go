package sessionbar

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/entities/sessionbar/clock"
)

type SessionBar struct {
	fonts *core.Fonts
	clock *clock.Clock
}

func New(cfg config.Timer, fonts *core.Fonts) *SessionBar {
	return &SessionBar{fonts: fonts, clock: clock.New(cfg, fonts)}
}

func (b *SessionBar) Close() {
	b.clock.Close()
}

func (b *SessionBar) Update(dt float32) {
	b.clock.Update(dt)
}

func (b *SessionBar) Draw() {
	screen := core.CurrentScreen()
	s := screen.Scale
	y := 42 * s
	accent := b.clock.Accent()
	done, goal := b.clock.SessionsToday(), b.clock.DailyGoal()

	label := fmt.Sprintf("Session %d / %d", done, goal)
	b.fonts.Body.DrawCentered(label, screen.Width/2, y, accent)

	bar := rl.Rectangle{X: screen.Width * 0.25, Y: y + 32*s, Width: screen.Width * 0.5, Height: 6 * s}
	ratio := min(float32(done)/float32(goal), 1)

	rl.DrawRectangleRounded(bar, 1, 6, core.ColorTrack)
	if ratio > 0 {
		bar.Width = max(bar.Width*ratio, bar.Height)
		rl.DrawRectangleRounded(bar, 1, 6, accent)
	}

	b.clock.Draw()
}
