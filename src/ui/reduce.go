package ui

import (
	"image/color"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type reduceAction int

const (
	reduceNone reduceAction = iota
	reduceCancel
	// reduceApply means the user confirmed; the picker's amount is the time to subtract.
	reduceApply
)

// reducePicker is the dialog for taking time off the running phase.
type reducePicker struct {
	// Up to four digits entered so far: tens-min, units-min, tens-sec, units-sec.
	digits []int
}

func (p *reducePicker) handle(k key) reduceAction {
	switch {
	case k.ch >= '0' && k.ch <= '9':
		d := int(k.ch - '0')
		// The tens-of-seconds digit must be 0–5.
		if len(p.digits) < 4 && (len(p.digits) != 2 || d <= 5) {
			p.digits = append(p.digits, d)
		}
	case k.code == rl.KeyBackspace:
		if len(p.digits) > 0 {
			p.digits = p.digits[:len(p.digits)-1]
		}
	case k.code == rl.KeyEnter || k.code == rl.KeyKpEnter:
		if len(p.digits) == 0 {
			return reduceCancel
		}
		return reduceApply
	case k.code == rl.KeyEscape:
		return reduceCancel
	}
	return reduceNone
}

func (p *reducePicker) amount() time.Duration {
	digit := func(i int) time.Duration {
		if i < len(p.digits) {
			return time.Duration(p.digits[i])
		}
		return 0
	}
	return (digit(0)*10+digit(1))*time.Minute + (digit(2)*10+digit(3))*time.Second
}

func (p *reducePicker) draw(f *fonts, w, h, s float32, col color.RGBA) {
	rl.DrawRectangle(0, 0, int32(w), int32(h), rl.Fade(colorBackground, 0.8))

	panel := rl.Rectangle{Width: 340 * s, Height: 190 * s}
	panel.X, panel.Y = (w-panel.Width)/2, (h-panel.Height)/2
	drawPanel(panel, s, col)

	cx := w / 2
	f.body.drawCentered("Reduce Remaining", cx, panel.Y+18*s, col)

	char := func(i int) byte {
		if i < len(p.digits) {
			return byte('0' + p.digits[i])
		}
		return '_'
	}
	display := string([]byte{char(0), char(1), ':', char(2), char(3)})
	f.body.drawCentered(display, cx, panel.Y+72*s, col)
	f.small.drawCentered("minutes : seconds", cx, panel.Y+104*s, colorDim)
	f.small.drawCentered("[Enter] Apply   [Esc] Cancel", cx, panel.Y+panel.Height-32*s, colorDim)
}

// drawPanel draws a dialog background with a border in the phase color.
func drawPanel(rec rl.Rectangle, s float32, col color.RGBA) {
	roundness := 16 * s / min(rec.Width, rec.Height)
	rl.DrawRectangleRounded(rec, roundness, 8, colorPanel)
	rl.DrawRectangleRoundedLinesEx(rec, roundness, 8, max(1.5*s, 1), col)
}
