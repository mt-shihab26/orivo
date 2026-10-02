package entities

import "github.com/mt-shihab26/orivo/src/core"

type Hints struct {
	world *core.World
}

func NewHints(world *core.World) *Hints {
	return &Hints{world: world}
}

func (h *Hints) Close() {}

func (h *Hints) Update(dt float32) {}

func (h *Hints) Draw() {
	w := h.world
	small := w.Fonts.Small

	text := "[Space] Toggle   [r] Reset   [n] Skip   [t] Todo   [T] Clear   [m] Millis   [d] Reduce"
	small.DrawCentered(small.Fit(text, w.Width-24*w.Scale), w.Width/2, w.Height-34*w.Scale, core.ColorDim)
}
