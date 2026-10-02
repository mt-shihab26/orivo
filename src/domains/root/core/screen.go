package core

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/systems/theme"
)

const (
	BaseWidth  = 720
	BaseHeight = 540
)

// The colors are read while drawing, so ApplyTheme must run on the render
// loop's goroutine.
var (
	ColorBackground = theme.Default.Background
	ColorPanel      = theme.Default.Panel
	ColorTrack      = theme.Default.Track
	ColorDim        = theme.Default.Dim
	ColorText       = theme.Default.Text

	ColorWork      = theme.Default.Work
	ColorBreak     = theme.Default.Break
	ColorLongBreak = theme.Default.LongBreak
)

func ApplyTheme(t theme.Theme) {
	ColorBackground, ColorPanel, ColorTrack, ColorDim, ColorText = t.Background, t.Panel, t.Track, t.Dim, t.Text
	ColorWork, ColorBreak, ColorLongBreak = t.Work, t.Break, t.LongBreak
}

type Screen struct {
	Width, Height, Scale float32
}

func CurrentScreen() Screen {
	width, height := float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())

	scale := min(width/BaseWidth, height/BaseHeight)
	scale = float32(int(scale*20)) / 20
	return Screen{Width: width, Height: height, Scale: min(max(scale, 0.5), 4)}
}
