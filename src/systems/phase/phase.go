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

func Named(name string) Phase {
	switch name {
	case "Break":
		return Break
	case "LongBreak":
		return LongBreak
	default:
		return Work
	}
}

func (p Phase) Name() string {
	switch p {
	case Break:
		return "Break"
	case LongBreak:
		return "LongBreak"
	default:
		return "Work"
	}
}

func (p Phase) Label() string {
	switch p {
	case Break:
		return "Short Break"
	case LongBreak:
		return "Long Break"
	default:
		return "Work Session"
	}
}

func (p Phase) Key() string {
	switch p {
	case Break:
		return "break"
	case LongBreak:
		return "long_break"
	default:
		return "work"
	}
}

func (p Phase) Duration(cfg config.Timer) time.Duration {
	switch p {
	case Break:
		return cfg.Break()
	case LongBreak:
		return cfg.LongBreak()
	default:
		return cfg.Work()
	}
}

func (p Phase) Color() color.RGBA {
	switch p {
	case Break:
		return color.RGBA{102, 187, 106, 255}
	case LongBreak:
		return color.RGBA{38, 198, 218, 255}
	default:
		return color.RGBA{239, 83, 80, 255}
	}
}

func (p Phase) EndMessage() (summary, body string) {
	switch p {
	case Break:
		return "Break Complete", "Ready to focus?"
	case LongBreak:
		return "Long Break Complete", "Ready to focus?"
	default:
		return "Work Session Complete", "Time for a break!"
	}
}
