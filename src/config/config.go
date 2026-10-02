// Package config loads the user's settings from ~/.config/orivo/config.toml.
package config

import (
	"errors"
	"io/fs"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	// ShowFPS shows the FPS counter on startup; toggle at runtime with Ctrl+F.
	ShowFPS bool `toml:"show_fps"`
	// Font is the path to a .ttf/.otf file; empty picks the system monospace font.
	Font  string `toml:"font"`
	Timer Timer  `toml:"timer"`
}

// Timer holds the pomodoro settings. Durations are in minutes as written in
// the config file; use the methods to get clamped values.
type Timer struct {
	ShowMillis        bool `toml:"show_millis"`
	WorkDuration      int  `toml:"work_duration"`
	BreakDuration     int  `toml:"break_duration"`
	LongBreakDuration int  `toml:"long_break_duration"`
	LongBreakInterval int  `toml:"long_break_interval"`
	DailySessionGoal  int  `toml:"daily_session_goal"`
}

func Default() Config {
	return Config{
		Timer: Timer{
			WorkDuration:      25,
			BreakDuration:     5,
			LongBreakDuration: 15,
			LongBreakInterval: 4,
			DailySessionGoal:  16,
		},
	}
}

// Load reads the config at path, returning the defaults if it does not exist.
func Load(path string) (Config, error) {
	cfg := Default()
	if _, err := toml.DecodeFile(path, &cfg); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Default(), err
	}
	return cfg, nil
}

// Work is the work session length, clamped to 1–120 min.
func (t Timer) Work() time.Duration {
	return minutes(t.WorkDuration, 1, 120)
}

// Break is the short break length, clamped to 1–60 min.
func (t Timer) Break() time.Duration {
	return minutes(t.BreakDuration, 1, 60)
}

// LongBreak is the long break length, clamped to 1–60 min.
func (t Timer) LongBreak() time.Duration {
	return minutes(t.LongBreakDuration, 1, 60)
}

// Interval is the number of work sessions between long breaks, clamped to 1–10.
func (t Timer) Interval() int {
	return clamp(t.LongBreakInterval, 1, 10)
}

// Goal is the daily session goal, clamped to 1–24.
func (t Timer) Goal() int {
	return clamp(t.DailySessionGoal, 1, 24)
}

func minutes(n, lo, hi int) time.Duration {
	return time.Duration(clamp(n, lo, hi)) * time.Minute
}

func clamp(n, lo, hi int) int {
	return min(max(n, lo), hi)
}
