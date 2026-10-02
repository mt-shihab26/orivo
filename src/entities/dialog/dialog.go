package dialog

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
)

type Dialog struct {
	Fonts  *core.Fonts
	title  string
	hint   string
	Accent color.RGBA
	Panel  rl.Rectangle
	open   bool
}

func New(fonts *core.Fonts, title, hint string) Dialog {
	return Dialog{Fonts: fonts, title: title, hint: hint}
}

func (d *Dialog) Close() {}

func (d *Dialog) Show() {
	d.open = true
}

func (d *Dialog) Hide() {
	d.open = false
}

func (d *Dialog) IsOpen() bool {
	return d.open
}

func (d *Dialog) Update(dt float32) {
	if d.open && rl.IsKeyPressed(rl.KeyEscape) {
		d.open = false
	}
}

func (d *Dialog) Draw() {
	if !d.open {
		return
	}
	screen := core.CurrentScreen()
	s := screen.Scale

	rl.DrawRectangle(0, 0, int32(screen.Width), int32(screen.Height), rl.Fade(core.ColorBackground, 0.8))

	roundness := 16 * s / min(d.Panel.Width, d.Panel.Height)
	rl.DrawRectangleRounded(d.Panel, roundness, 8, core.ColorPanel)
	rl.DrawRectangleRoundedLinesEx(d.Panel, roundness, 8, max(1.5*s, 1), d.Accent)

	cx := screen.Width / 2
	d.Fonts.Body.DrawCentered(d.title, cx, d.Panel.Y+16*s, d.Accent)
	d.Fonts.Small.DrawCentered(d.hint, cx, d.Panel.Y+d.Panel.Height-30*s, core.ColorDim)
}
