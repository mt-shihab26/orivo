package reducedialog

import (
	"image/color"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/entities/dialog"
)

type Timer interface {
	Accent() color.RGBA
	Reduce(d time.Duration)
}

type ReduceDialog struct {
	dialog.Dialog
	timer  Timer
	digits []int
}

func New(fonts *core.Fonts, timer Timer) *ReduceDialog {
	return &ReduceDialog{
		Dialog: dialog.New(fonts, "Reduce Remaining", "[Enter] Apply   [Esc] Cancel"),
		timer:  timer,
	}
}

func (d *ReduceDialog) Update(dt float32) {
	if !d.IsOpen() {
		if rl.IsKeyPressed(rl.KeyD) {
			d.Show()
			d.digits = nil
		}
		return
	}

	d.Dialog.Update(dt)
	if !d.IsOpen() {
		return
	}

	for digit := range int32(10) {
		if !rl.IsKeyPressed(rl.KeyZero+digit) && !rl.IsKeyPressed(rl.KeyKp0+digit) {
			continue
		}
		if len(d.digits) < 4 && (len(d.digits) != 2 || digit <= 5) {
			d.digits = append(d.digits, int(digit))
		}
	}

	if rl.IsKeyPressed(rl.KeyBackspace) || rl.IsKeyPressedRepeat(rl.KeyBackspace) {
		if len(d.digits) > 0 {
			d.digits = d.digits[:len(d.digits)-1]
		}
	}

	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		if len(d.digits) > 0 {
			padded := [4]time.Duration{}
			for i, digit := range d.digits {
				padded[i] = time.Duration(digit)
			}
			minutes := padded[0]*10 + padded[1]
			seconds := padded[2]*10 + padded[3]
			d.timer.Reduce(minutes*time.Minute + seconds*time.Second)
		}
		d.Hide()
	}
}

func (d *ReduceDialog) Draw() {
	if !d.IsOpen() {
		return
	}
	screen := core.CurrentScreen()
	s := screen.Scale

	d.Accent = d.timer.Accent()
	d.Panel = rl.Rectangle{Width: 340 * s, Height: 190 * s}
	d.Panel.X, d.Panel.Y = (screen.Width-d.Panel.Width)/2, (screen.Height-d.Panel.Height)/2
	d.Dialog.Draw()

	char := func(i int) byte {
		if i < len(d.digits) {
			return byte('0' + d.digits[i])
		}
		return '_'
	}
	display := string([]byte{char(0), char(1), ':', char(2), char(3)})

	cx := screen.Width / 2
	d.Fonts.Body.DrawCentered(display, cx, d.Panel.Y+72*s, d.Accent)
	d.Fonts.Small.DrawCentered("minutes : seconds", cx, d.Panel.Y+104*s, core.ColorDim)
}
