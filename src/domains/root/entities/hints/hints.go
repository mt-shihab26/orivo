package hints

import "orivo/src/domains/root/core"

type Hints struct {
	fonts *core.Fonts
}

func New(fonts *core.Fonts) *Hints {
	return &Hints{fonts: fonts}
}

func (h *Hints) Close() {}

func (h *Hints) Update(dt float32) {}

func (h *Hints) Draw() {
	screen := core.CurrentScreen()
	s := screen.Scale
	small := h.fonts.Small

	text := "[Space] Toggle   [r] Reset   [n] Skip   [t] Todo   [T] Clear   [m] Millis   [d] Reduce"
	small.DrawCentered(small.Fit(text, screen.Width-24*s), screen.Width/2, screen.Height-34*s, core.ColorDim)
}
