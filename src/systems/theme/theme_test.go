package theme

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

// writeTheme puts colors in dir/name/colors.toml, the layout Omarchy uses.
func writeTheme(t *testing.T, dir, name, colors string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name, "colors.toml"), []byte(colors), 0o644); err != nil {
		t.Fatal(err)
	}
}

const tokyoNight = `
mode = "dark"
accent = "#7aa2f7"
selection = "#292e42"
background = "#1a1b26"
lighter_background = "#24283b"
foreground = "#a9b1d6"
dark_foreground = "#565f89"
red = "#f7768e"
green = "#9ece6a"
cyan = "#449dab"
`

func TestParseHex(t *testing.T) {
	for _, tc := range []struct {
		text string
		want color.RGBA
		ok   bool
	}{
		{"#1a1b26", color.RGBA{0x1a, 0x1b, 0x26, 255}, true},
		{"F7768E", color.RGBA{0xf7, 0x76, 0x8e, 255}, true},
		{" #449dab ", color.RGBA{0x44, 0x9d, 0xab, 255}, true},
		{"#fff", color.RGBA{}, false},
		{"#12345g", color.RGBA{}, false},
		{"", color.RGBA{}, false},
	} {
		got, ok := ParseHex(tc.text)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseHex(%q) = %v, %v; want %v, %v", tc.text, got, ok, tc.want, tc.ok)
		}
	}
}

func TestLoadMapsTheOmarchyKeys(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "theme", tokyoNight)

	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := Theme{
		Background: color.RGBA{0x1a, 0x1b, 0x26, 255},
		Panel:      color.RGBA{0x24, 0x28, 0x3b, 255},
		Track:      color.RGBA{0x29, 0x2e, 0x42, 255},
		Dim:        color.RGBA{0x56, 0x5f, 0x89, 255},
		Text:       color.RGBA{0xa9, 0xb1, 0xd6, 255},
		Work:       color.RGBA{0xf7, 0x76, 0x8e, 255},
		Break:      color.RGBA{0x9e, 0xce, 0x6a, 255},
		LongBreak:  color.RGBA{0x44, 0x9d, 0xab, 255},
	}
	if got != want {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestLoadWithoutOmarchyGivesTheDefault(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "missing"))
	if err != nil || got != Default {
		t.Fatalf("got %+v, %v; want the default and no error", got, err)
	}
}

func TestMissingAndInvalidKeysKeepTheirDefault(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "theme", `
background = "#1a1b26"
red = "not a color"
green = 42
`)

	got, err := Load(dir)
	if err == nil {
		t.Error("invalid colors were not reported")
	}
	if got.Background != (color.RGBA{0x1a, 0x1b, 0x26, 255}) {
		t.Errorf("background = %v, want the theme's", got.Background)
	}
	if got.Work != Default.Work || got.Break != Default.Break {
		t.Errorf("invalid keys: work = %v, break = %v; want the defaults", got.Work, got.Break)
	}
	if got.Text != Default.Text || got.LongBreak != Default.LongBreak {
		t.Errorf("missing keys: text = %v, long break = %v; want the defaults", got.Text, got.LongBreak)
	}
}

func TestUnparsableFileGivesTheDefault(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "theme", `background = "#1a1b26`)

	got, err := Load(dir)
	if err == nil || got != Default {
		t.Fatalf("got %+v, %v; want the default and an error", got, err)
	}
}
