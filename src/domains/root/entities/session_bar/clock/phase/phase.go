package phase

import (
	"image/color"
	"time"

	"orivo/src/systems/config"
)

type Phase int

const (
	Work Phase = iota
	Break
	LongBreak
)

var phases = [...]struct {
	name, key, label string
	summary, body    string
	color            color.RGBA
	duration         func(config.Timer) time.Duration
}{
	Work:      {"Work", "work", "Work Session", "Work Session Complete", "Time for a break!", color.RGBA{239, 83, 80, 255}, config.Timer.Work},
	Break:     {"Break", "break", "Short Break", "Break Complete", "Ready to focus?", color.RGBA{102, 187, 106, 255}, config.Timer.Break},
	LongBreak: {"LongBreak", "long_break", "Long Break", "Long Break Complete", "Ready to focus?", color.RGBA{38, 198, 218, 255}, config.Timer.LongBreak},
}

func Named(name string) Phase {
	for p, info := range phases {
		if info.name == name {
			return Phase(p)
		}
	}
	return Work
}

func (p Phase) Name() string {
	return phases[p].name
}

func (p Phase) Label() string {
	return phases[p].label
}

func (p Phase) Key() string {
	return phases[p].key
}

func (p Phase) Duration(cfg config.Timer) time.Duration {
	return phases[p].duration(cfg)
}

func (p Phase) Color() color.RGBA {
	return phases[p].color
}

func (p Phase) EndMessage() (summary, body string) {
	return phases[p].summary, phases[p].body
}
