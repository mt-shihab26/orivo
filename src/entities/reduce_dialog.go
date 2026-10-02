package entities

import (
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
)

type ReduceDialog struct {
	input    *core.Input
	fonts    *core.Fonts
	pomodoro *Pomodoro
	open     bool
	digits   []int
}

func NewReduceDialog(input *core.Input, fonts *core.Fonts, pomodoro *Pomodoro) *ReduceDialog {
	return &ReduceDialog{input: input, fonts: fonts, pomodoro: pomodoro}
}

func (d *ReduceDialog) Close() {}

func (d *ReduceDialog) IsOpen() bool {
	return d.open
}

func (d *ReduceDialog) Update(dt float32) {
	for _, key := range d.input.Keys {
		switch {
		case !d.open:
			if key.Ch == 'd' {
				d.open = true
				d.digits = nil
			}
		case key.Ch >= '0' && key.Ch <= '9':
			digit := int(key.Ch - '0')
			if len(d.digits) < 4 && (len(d.digits) != 2 || digit <= 5) {
				d.digits = append(d.digits, digit)
			}
		case key.Code == rl.KeyBackspace:
			if len(d.digits) > 0 {
				d.digits = d.digits[:len(d.digits)-1]
			}
		case key.Code == rl.KeyEnter || key.Code == rl.KeyKpEnter:
			if len(d.digits) > 0 {
				d.pomodoro.Reduce(d.amount())
			}
			d.open = false
		case key.Code == rl.KeyEscape:
			d.open = false
		}
	}
}

func (d *ReduceDialog) amount() time.Duration {
	digit := func(i int) time.Duration {
		if i < len(d.digits) {
			return time.Duration(d.digits[i])
		}
		return 0
	}
	return (digit(0)*10+digit(1))*time.Minute + (digit(2)*10+digit(3))*time.Second
}

func (d *ReduceDialog) Draw() {
	if !d.open {
		return
	}
	screen := core.CurrentScreen()
	s := screen.Scale
	fonts := d.fonts
	accent := d.pomodoro.Phase.Color()

	panel := rl.Rectangle{Width: 340 * s, Height: 190 * s}
	panel.X, panel.Y = (screen.Width-panel.Width)/2, (screen.Height-panel.Height)/2
	screen.DrawDialog(panel, accent)

	cx := screen.Width / 2
	fonts.Body.DrawCentered("Reduce Remaining", cx, panel.Y+18*s, accent)

	char := func(i int) byte {
		if i < len(d.digits) {
			return byte('0' + d.digits[i])
		}
		return '_'
	}
	display := string([]byte{char(0), char(1), ':', char(2), char(3)})
	fonts.Body.DrawCentered(display, cx, panel.Y+72*s, accent)
	fonts.Small.DrawCentered("minutes : seconds", cx, panel.Y+104*s, core.ColorDim)
	fonts.Small.DrawCentered("[Enter] Apply   [Esc] Cancel", cx, panel.Y+panel.Height-32*s, core.ColorDim)
}
