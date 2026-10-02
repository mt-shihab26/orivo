package core

import (
	"image/color"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/config"
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
	Config config.Config
	Fonts  *Fonts

	Width, Height, Scale float32
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
}

func (w *World) DrawDialog(panel rl.Rectangle) {
	rl.DrawRectangle(0, 0, int32(w.Width), int32(w.Height), rl.Fade(ColorBackground, 0.8))

	roundness := 16 * w.Scale / min(panel.Width, panel.Height)
	rl.DrawRectangleRounded(panel, roundness, 8, ColorPanel)
	rl.DrawRectangleRoundedLinesEx(panel, roundness, 8, max(1.5*w.Scale, 1), w.Color)
}
