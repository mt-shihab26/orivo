package core

import (
	"image/color"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/phase"
	"github.com/mt-shihab26/orivo/src/sessions"
	"github.com/mt-shihab26/orivo/src/timer"
	"github.com/mt-shihab26/orivo/src/todos"
)

const (
	BaseWidth  = 720
	BaseHeight = 540
)

var (
	ColorBackground = color.RGBA{15, 17, 21, 255}
	ColorPanel      = color.RGBA{23, 26, 33, 255}
	ColorTrack      = color.RGBA{42, 46, 55, 255}
	ColorDim        = color.RGBA{107, 114, 128, 255}
	ColorText       = color.RGBA{229, 231, 235, 255}
)

type World struct {
	Config   config.Config
	Timer    *timer.State
	Todos    todos.Source
	Sessions *sessions.Log
	Fonts    *Fonts

	Width, Height, Scale float32
	Snap                 timer.Snapshot
	Color                color.RGBA
	Keys                 []Key

	quit atomic.Bool
}

func (w *World) Quit() {
	w.quit.Store(true)
}

func (w *World) ShouldQuit() bool {
	return w.quit.Load()
}

func (w *World) Sync() {
	w.Width, w.Height = float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())

	scale := min(w.Width/BaseWidth, w.Height/BaseHeight)
	scale = float32(int(scale*20)) / 20
	w.Scale = min(max(scale, 0.5), 4)
	w.Fonts.Ensure(w.Scale)

	w.Snap = w.Timer.Snapshot()
	w.Color = PhaseColor(w.Snap.Phase)
}

func PhaseColor(p phase.Phase) color.RGBA {
	switch p {
	case phase.Break:
		return color.RGBA{102, 187, 106, 255}
	case phase.LongBreak:
		return color.RGBA{38, 198, 218, 255}
	default:
		return color.RGBA{239, 83, 80, 255}
	}
}

func (w *World) DrawDialog(panel rl.Rectangle) {
	rl.DrawRectangle(0, 0, int32(w.Width), int32(w.Height), rl.Fade(ColorBackground, 0.8))

	roundness := 16 * w.Scale / min(panel.Width, panel.Height)
	rl.DrawRectangleRounded(panel, roundness, 8, ColorPanel)
	rl.DrawRectangleRoundedLinesEx(panel, roundness, 8, max(1.5*w.Scale, 1), w.Color)
}
