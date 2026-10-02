package logx

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mt-shihab26/orivo/src/systems/paths"
)

const maxBytes = 5 * 1024 * 1024

var mu sync.Mutex

func Error(format string, args ...any) { write("ERROR", format, args...) }
func Warn(format string, args ...any)  { write("WARN", format, args...) }
func Info(format string, args ...any)  { write("INFO", format, args...) }

func write(level, format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()

	path := paths.Log()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)

	if info, err := os.Stat(path); err == nil && info.Size() >= maxBytes {
		_ = os.Rename(path, path+".1")
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()

	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	fmt.Fprintf(file, "[%s] %s: %s\n", ts, level, fmt.Sprintf(format, args...))
}
