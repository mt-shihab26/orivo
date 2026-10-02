package config

import (
	"errors"
	"io/fs"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ShowFPS bool   `toml:"show_fps"`
	Font    string `toml:"font"`
	Timer   Timer  `toml:"timer"`
}

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

func Load(path string) (Config, error) {
	cfg := Default()
	if _, err := toml.DecodeFile(path, &cfg); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Default(), err
	}
	return cfg, nil
}

func (t Timer) Work() time.Duration {
	return minutes(t.WorkDuration, 1, 120)
}

func (t Timer) Break() time.Duration {
	return minutes(t.BreakDuration, 1, 60)
}

func (t Timer) LongBreak() time.Duration {
	return minutes(t.LongBreakDuration, 1, 60)
}

func (t Timer) Interval() int {
	return clamp(t.LongBreakInterval, 1, 10)
}

func (t Timer) Goal() int {
	return clamp(t.DailySessionGoal, 1, 24)
}

func minutes(n, lo, hi int) time.Duration {
	return time.Duration(clamp(n, lo, hi)) * time.Minute
}

func clamp(n, lo, hi int) int {
	return min(max(n, lo), hi)
}
