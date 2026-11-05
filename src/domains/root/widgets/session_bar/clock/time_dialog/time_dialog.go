package time_dialog

import (
	"image/color"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/domains/root/core"
	"orivo/src/domains/root/widgets/dialog"
)

type Timer interface {
	Accent() color.RGBA
}

// TimeDialog asks for a time as mm:ss and hands it to apply, to take off the
// timer or to give back to it.
type TimeDialog struct {
	dialog.Dialog
	timer  Timer
	key    int32
	apply  func(time.Duration)
	digits []int
}

// New makes a dialog titled title that key opens and that hands the time
// entered to apply.
func New(fonts *core.Fonts, timer Timer, title string, key int32, apply func(time.Duration)) *TimeDialog {
	return &TimeDialog{
		Dialog: dialog.New(fonts, title, ""),
		timer:  timer,
		key:    key,
		apply:  apply,
	}
}

// Open shows the dialog with nothing entered.
func (d *TimeDialog) Open() {
	d.Show()
	d.digits = nil
}

func (d *TimeDialog) Update(dt float32) {
	if !d.IsOpen() {
		if rl.IsKeyPressed(d.key) {
			d.Open()
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
		d.submit()
	}
}

// submit hands the entered time to apply, if any, and closes.
func (d *TimeDialog) submit() {
	if len(d.digits) > 0 {
		d.apply(entered(d.digits))
	}
	d.Hide()
}

// entered reads the digits typed so far as mm:ss, the missing ones as 0.
func entered(digits []int) time.Duration {
	padded := [4]time.Duration{}
	for i, digit := range digits {
		padded[i] = time.Duration(digit)
	}
	minutes := padded[0]*10 + padded[1]
	seconds := padded[2]*10 + padded[3]
	return minutes*time.Minute + seconds*time.Second
}

func (d *TimeDialog) Draw() {
	if !d.IsOpen() {
		return
	}
	screen := core.CurrentScreen()
	s := screen.Scale

	d.Accent = d.timer.Accent()
	d.Panel = rl.Rectangle{Width: 340 * s, Height: 200 * s}
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
	d.Fonts.Body.DrawCentered(display, cx, d.Panel.Y+68*s, d.Accent)
	d.Fonts.Small.DrawCentered("minutes : seconds", cx, d.Panel.Y+100*s, core.ColorDim)

	switch d.Buttons("[Enter] Apply", "[Esc] Cancel") {
	case 0:
		d.submit()
	case 1:
		d.Hide()
	}
}
