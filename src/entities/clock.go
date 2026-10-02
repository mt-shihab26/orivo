package entities

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
)

type Clock struct {
	fonts *core.Fonts
	timer core.Timer
}

func NewClock(fonts *core.Fonts, timer core.Timer) *Clock {
	return &Clock{fonts: fonts, timer: timer}
}

func (c *Clock) Close() {}

func (c *Clock) Update(dt float32) {}

func (c *Clock) Draw() {
	screen := core.CurrentScreen()
	s := screen.Scale
	timer := c.timer
	fonts := c.fonts
	accent := timer.Accent()
	remaining := timer.Remaining()

	top, bottom := 92*s, screen.Height-96*s
	radius := max(min((bottom-top)/2-6*s, screen.Width*0.38), 40*s)
	cx, cy := screen.Width/2, (top+bottom)/2
	center := rl.Vector2{X: cx, Y: cy}

	thickness := max(7*s, 3)
	rl.DrawRing(center, radius-thickness, radius, 0, 360, 96, core.ColorTrack)
	elapsed := 1 - float32(remaining)/float32(timer.Total())
	if elapsed > 0 {
		rl.DrawRing(center, radius-thickness, radius, -90, -90+360*min(elapsed, 1), 96, accent)
	}

	text := clockText(remaining, timer.ShowMillis())
	inner := (radius - thickness) * 2 * 0.76
	fonts.EnsureClock(min(inner/(0.6*float32(len(text))), radius*0.62))

	digits := fonts.Clock
	if width := digits.Width(text); width > inner {
		digits.Size *= inner / width
	}
	digits.DrawCentered(text, cx, cy-digits.Size/2, accent)

	fonts.Body.DrawCentered(timer.PhaseLabel(), cx, cy-digits.Size/2-fonts.Body.Size-10*s, accent)

	status, statusColor := "Paused", core.ColorDim
	if timer.Running() {
		status, statusColor = "Running", accent
	}
	fonts.Body.DrawCentered(status, cx, cy+digits.Size/2+10*s, statusColor)
}

func clockText(remaining time.Duration, showMillis bool) string {
	if showMillis {
		millis := remaining.Milliseconds()
		return fmt.Sprintf("%02d:%02d.%02d", millis/60000, millis/1000%60, millis%1000/10)
	}
	secs := int64((remaining + time.Second - 1) / time.Second)
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}
