package phase

import (
	"fmt"
	"time"

	"github.com/mt-shihab26/orivo/src/config"
)

type Phase int

const (
	Work Phase = iota
	Break
	LongBreak
)

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

func (p Phase) MarshalText() ([]byte, error) {
	switch p {
	case Break:
		return []byte("Break"), nil
	case LongBreak:
		return []byte("LongBreak"), nil
	default:
		return []byte("Work"), nil
	}
}

func (p *Phase) UnmarshalText(text []byte) error {
	switch string(text) {
	case "Work":
		*p = Work
	case "Break":
		*p = Break
	case "LongBreak":
		*p = LongBreak
	default:
		return fmt.Errorf("unknown phase %q", text)
	}
	return nil
}
