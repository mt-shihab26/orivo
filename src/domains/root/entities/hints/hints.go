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

	lines := []string{
		"[Space] Toggle   [r] Reset   [n] Skip   [m] Millis   [d] Reduce",
		"[t] Todo   [T] Clear   [s] Sync",
	}
	for i, text := range lines {
		y := screen.Height - 50*s + float32(i)*20*s
		small.DrawCentered(small.Fit(text, screen.Width-24*s), screen.Width/2, y, core.ColorDim)
	}
}
