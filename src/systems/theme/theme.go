package theme

import (
	"errors"
	"fmt"
	"image/color"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Theme struct {
	Background color.RGBA
	Panel      color.RGBA
	Track      color.RGBA
	Dim        color.RGBA
	Text       color.RGBA
	Work       color.RGBA
	Break      color.RGBA
	LongBreak  color.RGBA
}

// Default is orivo's own palette, used for anything the Omarchy theme does
// not give.
var Default = Theme{
	Background: color.RGBA{15, 17, 21, 255},
	Panel:      color.RGBA{23, 26, 33, 255},
	Track:      color.RGBA{42, 46, 55, 255},
	Dim:        color.RGBA{107, 114, 128, 255},
	Text:       color.RGBA{229, 231, 235, 255},
	Work:       color.RGBA{239, 83, 80, 255},
	Break:      color.RGBA{102, 187, 106, 255},
	LongBreak:  color.RGBA{38, 198, 218, 255},
}

// OmarchyDir is where Omarchy keeps the current theme, as theme/colors.toml,
// and its name, as theme.name.
func OmarchyDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local/state/omarchy/current")
}

func colorsPath(dir string) string {
	return filepath.Join(dir, "theme", "colors.toml")
}

// Load reads the Omarchy theme in dir. A missing file gives Default with no
// error; an unreadable one gives Default and the error, and a missing or
// invalid key keeps that one color's default.
func Load(dir string) (Theme, error) {
	t := Default

	var keys map[string]any
	_, err := toml.DecodeFile(colorsPath(dir), &keys)
	if errors.Is(err, fs.ErrNotExist) {
		return t, nil
	}
	if err != nil {
		return t, err
	}

	var bad []string
	for key, target := range map[string]*color.RGBA{
		"background":         &t.Background,
		"lighter_background": &t.Panel,
		"selection":          &t.Track,
		"dark_foreground":    &t.Dim,
		"foreground":         &t.Text,
		"red":                &t.Work,
		"green":              &t.Break,
		"cyan":               &t.LongBreak,
	} {
		value, ok := keys[key]
		if !ok {
			continue
		}
		text, _ := value.(string)
		parsed, ok := ParseHex(text)
		if !ok {
			bad = append(bad, fmt.Sprintf("%s = %v", key, value))
			continue
		}
		*target = parsed
	}
	if len(bad) > 0 {
		return t, fmt.Errorf("invalid colors in %s: %s", colorsPath(dir), strings.Join(bad, ", "))
	}
	return t, nil
}

// ParseHex reads "#rrggbb", with or without the "#".
func ParseHex(text string) (color.RGBA, bool) {
	text = strings.TrimPrefix(strings.TrimSpace(text), "#")
	if len(text) != 6 {
		return color.RGBA{}, false
	}
	n, err := strconv.ParseUint(text, 16, 32)
	if err != nil {
		return color.RGBA{}, false
	}
	return color.RGBA{uint8(n >> 16), uint8(n >> 8), uint8(n), 255}, true
}
