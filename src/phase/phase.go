// Package phase defines the phases of the pomodoro cycle.
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
	// LongBreak is the longer rest after a full interval of work sessions.
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

// Key is the identifier used in the session log and over IPC.
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

// Duration is the configured length of the phase.
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

// MarshalText writes the name store.json has always used for the phase, which
// external readers of that file (e.g. the bar widget) depend on.
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
