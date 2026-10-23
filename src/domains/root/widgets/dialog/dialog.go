package dialog

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/domains/root/core"
)

type Dialog struct {
	Fonts  *core.Fonts
	title  string
	hint   string
	Accent color.RGBA
	Panel  rl.Rectangle
	open   bool
	// Set on the frame the dialog opens, which may be the frame of the click
	// that opened it; that click must not land on the dialog too.
	fresh bool
}

func New(fonts *core.Fonts, title, hint string) Dialog {
	return Dialog{Fonts: fonts, title: title, hint: hint}
}

func (d *Dialog) Attach() {}

func (d *Dialog) Detach() {
	d.Hide()
}

func (d *Dialog) Show() {
	if !d.open {
		core.OpenModal()
	}
	d.open, d.fresh = true, true
}

func (d *Dialog) Hide() {
	if d.open {
		core.CloseModal()
	}
	d.open = false
}

func (d *Dialog) IsOpen() bool {
	return d.open
}

func (d *Dialog) Update(dt float32) {
	if d.open && rl.IsKeyPressed(rl.KeyEscape) {
		d.Hide()
	}
}

func (d *Dialog) Draw() {
	if !d.open {
		return
	}
	screen := core.CurrentScreen()
	s := screen.Scale

	if !d.fresh {
		core.Unlock()
	}
	d.fresh = false

	rl.DrawRectangle(0, 0, int32(screen.Width), int32(screen.Height), rl.Fade(core.ColorBackground, 0.8))

	roundness := 16 * s / min(d.Panel.Width, d.Panel.Height)
	rl.DrawRectangleRounded(d.Panel, roundness, 8, core.ColorPanel)
	rl.DrawRectangleRoundedLinesEx(d.Panel, roundness, 8, max(1.5*s, 1), d.Accent)

	cx := screen.Width / 2
	d.Fonts.Body.DrawCentered(d.title, cx, d.Panel.Y+16*s, d.Accent)
}

// Buttons draws the hint and a row of buttons at the foot of the panel, and
// returns the index of the one clicked, or -1.
func (d *Dialog) Buttons(labels ...string) int {
	s := core.CurrentScreen().Scale
	gap, height := 8*s, 24*s

	width := float32(0)
	if d.hint != "" {
		width += d.Fonts.Small.Width(d.hint) + 2*gap
	}
	for _, label := range labels {
		width += core.ButtonWidth(d.Fonts, label) + gap
	}
	x := d.Panel.X + (d.Panel.Width-width+gap)/2
	y := d.Panel.Y + d.Panel.Height - height - 12*s

	if d.hint != "" {
		d.Fonts.Small.Draw(d.hint, x, y+(height-d.Fonts.Small.Size)/2, core.ColorDim)
		x += d.Fonts.Small.Width(d.hint) + 2*gap
	}
	clicked := -1
	for i, label := range labels {
		w := core.ButtonWidth(d.Fonts, label)
		if core.Button(rl.Rectangle{X: x, Y: y, Width: w, Height: height}, label) {
			clicked = i
		}
		x += w + gap
	}
	return clicked
}
