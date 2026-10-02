package core

import (
	"image/color"
	"time"
)

type Stat struct {
	Sessions int
	Secs     int
}

type Timer interface {
	PhaseLabel() string
	Accent() color.RGBA
	Running() bool
	ShowMillis() bool
	Remaining() time.Duration
	Total() time.Duration
	SessionsToday() int
	DailyGoal() int
	TodoID() string
	TodoText() string
	Stat(todoID string) Stat
	SetTodo(id, text string)
	Reduce(d time.Duration)
}
