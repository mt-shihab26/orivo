package logx

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"orivo/src/systems/config"
)

// Each week gets its own file, named after the Saturday it starts on, and
// only the current week's is kept.
const weekStartsOn = time.Saturday

var (
	mu      sync.Mutex
	current string
)

func Error(format string, args ...any) { write("ERROR", format, args...) }
func Warn(format string, args ...any)  { write("WARN", format, args...) }
func Info(format string, args ...any)  { write("INFO", format, args...) }

func write(level, format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	dir := config.LogDir()
	path := filepath.Join(dir, fileFor(now))
	_ = os.MkdirAll(dir, 0o700)

	if path != current {
		current = path
		prune(dir, filepath.Base(path))
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()

	ts := now.UTC().Format("2006-01-02T15:04:05Z")
	fmt.Fprintf(file, "[%s] %s: %s\n", ts, level, fmt.Sprintf(format, args...))
}

// fileFor names the log for the week that t falls in.
func fileFor(t time.Time) string {
	return "orivo-" + weekStart(t).Format(time.DateOnly) + ".log"
}

func weekStart(t time.Time) time.Time {
	t = t.Local()
	back := (int(t.Weekday()) - int(weekStartsOn) + 7) % 7
	y, m, d := t.Date()
	return time.Date(y, m, d-back, 0, 0, 0, 0, time.Local)
}

// prune removes every log in dir but keep, including the single orivo.log
// older versions wrote.
func prune(dir, keep string) {
	old, _ := filepath.Glob(filepath.Join(dir, "orivo-*.log"))
	old = append(old, filepath.Join(dir, "orivo.log"), filepath.Join(dir, "orivo.log.1"))
	for _, path := range old {
		if filepath.Base(path) != keep {
			_ = os.Remove(path)
		}
	}
}
