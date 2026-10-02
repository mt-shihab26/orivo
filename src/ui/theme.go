package ui

import (
	"image/color"
	"os/exec"
	"slices"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/logx"
	"github.com/mt-shihab26/orivo/src/phase"
)

var (
	colorBackground = color.RGBA{15, 17, 21, 255}
	colorPanel      = color.RGBA{23, 26, 33, 255}
	colorTrack      = color.RGBA{42, 46, 55, 255}
	colorDim        = color.RGBA{107, 114, 128, 255}
	colorText       = color.RGBA{229, 231, 235, 255}
)

func phaseColor(p phase.Phase) color.RGBA {
	switch p {
	case phase.Break:
		return color.RGBA{102, 187, 106, 255}
	case phase.LongBreak:
		return color.RGBA{38, 198, 218, 255}
	default:
		return color.RGBA{239, 83, 80, 255}
	}
}

// face is a font rasterised for one pixel size.
type face struct {
	font rl.Font
	size float32
	// Extra spacing between glyphs, only needed by raylib's built-in font.
	spacing float32
}

func (f face) width(text string) float32 {
	return rl.MeasureTextEx(f.font, text, f.size, f.spacing).X
}

func (f face) draw(text string, x, y float32, col color.RGBA) {
	// Whole pixels keep the glyphs crisp.
	pos := rl.Vector2{X: float32(int(x)), Y: float32(int(y))}
	rl.DrawTextEx(f.font, text, pos, f.size, f.spacing, col)
}

// drawCentered draws text horizontally centered on cx.
func (f face) drawCentered(text string, cx, y float32, col color.RGBA) {
	f.draw(text, cx-f.width(text)/2, y, col)
}

// fit shortens text with an ellipsis until it is at most maxWidth wide.
func (f face) fit(text string, maxWidth float32) string {
	if f.width(text) <= maxWidth {
		return text
	}
	runes := []rune(text)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if f.width(string(runes[:mid])+"…") <= maxWidth {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return strings.TrimRight(string(runes[:lo]), " ") + "…"
}

// fonts rasterises the UI font at the exact pixel sizes in use, reloading
// whenever the window scale or the set of needed glyphs changes — raylib
// bakes a font into a fixed-size atlas, and scaling that atlas looks blurry.
type fonts struct {
	// Font file path; empty falls back to raylib's built-in font.
	path string
	// Every glyph the text faces must contain, sorted.
	runes []rune

	small, body, clock face
	// Scale the text faces were loaded for, zero when they need (re)loading.
	scale float32
}

func newFonts(configured string) *fonts {
	f := &fonts{path: configured}
	if f.path == "" {
		f.path = systemFont()
	}

	// Printable ASCII and Latin-1, plus the punctuation the UI itself uses.
	for r := rune(32); r < 256; r++ {
		if r < 127 || r >= 160 {
			f.runes = append(f.runes, r)
		}
	}
	f.need("…—–•’‘“”")
	return f
}

// systemFont asks fontconfig for the default monospace font file.
func systemFont() string {
	out, err := exec.Command("fc-match", "--format=%{file}", "monospace").Output()
	if err != nil {
		logx.Warn("fonts: fc-match failed, using the built-in font: %v", err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

// need makes sure the text faces can draw every character of text.
func (f *fonts) need(text string) {
	for _, r := range text {
		if i, found := slices.BinarySearch(f.runes, r); !found {
			f.runes = slices.Insert(f.runes, i, r)
			f.scale = 0
		}
	}
}

// ensure loads the text faces for the given window scale.
func (f *fonts) ensure(scale float32) {
	if f.scale == scale {
		return
	}
	f.unload(f.small)
	f.unload(f.body)
	f.small = f.load(14*scale, f.runes)
	f.body = f.load(20*scale, f.runes)
	f.scale = scale
}

// ensureClock loads the clock face at the given pixel size.
func (f *fonts) ensureClock(size float32) {
	if f.clock.size == float32(int(size)) {
		return
	}
	f.unload(f.clock)
	f.clock = f.load(size, []rune("0123456789:."))
}

func (f *fonts) load(size float32, runes []rune) face {
	px := max(int32(size), 8)
	if f.path != "" {
		font := rl.LoadFontEx(f.path, px, runes)
		// A failed load hands back the built-in font instead.
		if font.Texture.ID != rl.GetFontDefault().Texture.ID {
			rl.SetTextureFilter(font.Texture, rl.FilterBilinear)
			return face{font: font, size: float32(px)}
		}
		logx.Warn("fonts: failed to load %s, using the built-in font", f.path)
		f.path = ""
	}
	return face{font: rl.GetFontDefault(), size: float32(px), spacing: float32(px) / 10}
}

func (f *fonts) unload(old face) {
	if old.font.Texture.ID != 0 && old.font.Texture.ID != rl.GetFontDefault().Texture.ID {
		rl.UnloadFont(old.font)
	}
}

func (f *fonts) close() {
	f.unload(f.small)
	f.unload(f.body)
	f.unload(f.clock)
}
