package entities

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
)

type Clock struct {
	world *core.World
}

func NewClock(world *core.World) *Clock {
	return &Clock{world: world}
}

func (c *Clock) Close() {}

func (c *Clock) Update(dt float32) {}

func (c *Clock) Draw() {
	w := c.world
	s := w.Scale
	snap := w.Snap
	fonts := w.Fonts

	top, bottom := 92*s, w.Height-96*s
	radius := max(min((bottom-top)/2-6*s, w.Width*0.38), 40*s)
	cx, cy := w.Width/2, (top+bottom)/2
	center := rl.Vector2{X: cx, Y: cy}

	thickness := max(7*s, 3)
	rl.DrawRing(center, radius-thickness, radius, 0, 360, 96, core.ColorTrack)
	elapsed := 1 - float32(snap.Remaining)/float32(snap.Total)
	if elapsed > 0 {
		rl.DrawRing(center, radius-thickness, radius, -90, -90+360*min(elapsed, 1), 96, w.Color)
	}

	text := clockText(snap.Remaining, snap.ShowMillis)
	inner := (radius - thickness) * 2 * 0.76
	fonts.EnsureClock(min(inner/(0.6*float32(len(text))), radius*0.62))

	digits := fonts.Clock
	if width := digits.Width(text); width > inner {
		digits.Size *= inner / width
	}
	digits.DrawCentered(text, cx, cy-digits.Size/2, w.Color)

	fonts.Body.DrawCentered(snap.Phase.Label(), cx, cy-digits.Size/2-fonts.Body.Size-10*s, w.Color)

	status, statusColor := "Paused", core.ColorDim
	if snap.Running {
		status, statusColor = "Running", w.Color
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
