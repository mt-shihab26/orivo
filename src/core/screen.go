package core

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
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

type Screen struct {
	Width, Height, Scale float32
}

func CurrentScreen() Screen {
	width, height := float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())

	scale := min(width/BaseWidth, height/BaseHeight)
	scale = float32(int(scale*20)) / 20
	return Screen{Width: width, Height: height, Scale: min(max(scale, 0.5), 4)}
}

func (s Screen) DrawDialog(panel rl.Rectangle, border color.RGBA) {
	rl.DrawRectangle(0, 0, int32(s.Width), int32(s.Height), rl.Fade(ColorBackground, 0.8))

	roundness := 16 * s.Scale / min(panel.Width, panel.Height)
	rl.DrawRectangleRounded(panel, roundness, 8, ColorPanel)
	rl.DrawRectangleRoundedLinesEx(panel, roundness, 8, max(1.5*s.Scale, 1), border)
}
