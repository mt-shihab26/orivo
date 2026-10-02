package core

import (
	"image/color"
	"os/exec"
	"slices"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/systems/logx"
)

type Face struct {
	font    rl.Font
	Size    float32
	spacing float32
}

func (f Face) Width(text string) float32 {
	return rl.MeasureTextEx(f.font, text, f.Size, f.spacing).X
}

func (f Face) Draw(text string, x, y float32, col color.RGBA) {
	pos := rl.Vector2{X: float32(int(x)), Y: float32(int(y))}
	rl.DrawTextEx(f.font, text, pos, f.Size, f.spacing, col)
}

func (f Face) DrawCentered(text string, cx, y float32, col color.RGBA) {
	f.Draw(text, cx-f.Width(text)/2, y, col)
}

func (f Face) Fit(text string, maxWidth float32) string {
	if f.Width(text) <= maxWidth {
		return text
	}
	runes := []rune(text)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if f.Width(string(runes[:mid])+"…") <= maxWidth {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return strings.TrimRight(string(runes[:lo]), " ") + "…"
}

type Fonts struct {
	path  string
	runes []rune

	Small, Body, Clock Face
	scale              float32
}

func NewFonts(configured string) *Fonts {
	f := &Fonts{path: configured}
	if f.path == "" {
		f.path = systemFont()
	}

	for r := rune(32); r < 256; r++ {
		if r < 127 || r >= 160 {
			f.runes = append(f.runes, r)
		}
	}
	f.Need("…—–•’‘“”")
	return f
}

func systemFont() string {
	out, err := exec.Command("fc-match", "--format=%{file}", "monospace").Output()
	if err != nil {
		logx.Warn("fonts: fc-match failed, using the built-in font: %v", err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (f *Fonts) Need(text string) {
	for _, r := range text {
		if i, found := slices.BinarySearch(f.runes, r); !found {
			f.runes = slices.Insert(f.runes, i, r)
			f.scale = 0
		}
	}
}

func (f *Fonts) Ensure(scale float32) {
	if f.scale == scale {
		return
	}
	f.unload(f.Small)
	f.unload(f.Body)
	f.Small = f.load(14*scale, f.runes)
	f.Body = f.load(20*scale, f.runes)
	f.scale = scale
}

func (f *Fonts) EnsureClock(size float32) {
	if f.Clock.Size == float32(max(int32(size), 8)) {
		return
	}
	f.unload(f.Clock)
	f.Clock = f.load(size, []rune("0123456789:."))
}

func (f *Fonts) load(size float32, runes []rune) Face {
	px := max(int32(size), 8)
	if f.path != "" {
		font := rl.LoadFontEx(f.path, px, runes)
		if font.Texture.ID != rl.GetFontDefault().Texture.ID {
			rl.SetTextureFilter(font.Texture, rl.FilterBilinear)
			return Face{font: font, Size: float32(px)}
		}
		logx.Warn("fonts: failed to load %s, using the built-in font", f.path)
		f.path = ""
	}
	return Face{font: rl.GetFontDefault(), Size: float32(px), spacing: float32(px) / 10}
}

func (f *Fonts) unload(old Face) {
	if old.font.Texture.ID != 0 && old.font.Texture.ID != rl.GetFontDefault().Texture.ID {
		rl.UnloadFont(old.font)
	}
}

func (f *Fonts) Close() {
	f.unload(f.Small)
	f.unload(f.Body)
	f.unload(f.Clock)
}
