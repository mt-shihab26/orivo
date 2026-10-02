package sessions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"orivo/src/systems/logx"
)

type Session struct {
	Phase        string    `json:"phase"`
	DurationSecs int       `json:"duration_secs"`
	StartedAt    time.Time `json:"started_at"`
	EndedAt      time.Time `json:"ended_at"`
	TodoID       string    `json:"todo_id,omitempty"`
	TodoText     string    `json:"todo_text,omitempty"`
}

type Stat struct {
	Sessions int
	Secs     int
}

// History keeps one file per day in dir, named after the date the sessions
// ended on, and counts only the day asked about, so every todo starts each
// day from zero.
type History struct {
	dir      string
	day      string
	sessions int
	stats    map[string]Stat
}

func Open(dir string) (*History, error) {
	return &History{dir: dir}, migrate(dir)
}

func (h *History) Record(s Session) error {
	day := dayOf(s.EndedAt)
	h.load(day)
	h.add(s)
	s.StartedAt = s.StartedAt.Round(0)
	s.EndedAt = s.EndedAt.Round(0)

	line, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(h.dir, 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(h.file(day), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(append(line, '\n'))
	return err
}

func (h *History) CountOn(day time.Time) int {
	h.load(dayOf(day))
	return h.sessions
}

func (h *History) StatOn(day time.Time, todoID string) Stat {
	h.load(dayOf(day))
	return h.stats[todoID]
}

// load reads day's file unless it is the one already counted.
func (h *History) load(day string) {
	if day == h.day {
		return
	}
	h.day, h.sessions, h.stats = day, 0, map[string]Stat{}

	file, err := os.Open(h.file(day))
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		logx.Error("failed to read %s: %v", h.file(day), err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var s Session
		if json.Unmarshal(scanner.Bytes(), &s) == nil {
			h.add(s)
		}
	}
	if err := scanner.Err(); err != nil {
		logx.Error("failed to read %s: %v", h.file(day), err)
	}
}

func (h *History) add(s Session) {
	if s.Phase != "work" {
		return
	}
	h.sessions++
	if s.TodoID != "" {
		stat := h.stats[s.TodoID]
		stat.Sessions++
		stat.Secs += s.DurationSecs
		h.stats[s.TodoID] = stat
	}
}

func (h *History) file(day string) string {
	return filepath.Join(h.dir, day+".jsonl")
}

// migrate splits the single dir.jsonl older versions wrote into one file per
// day. The files are built aside and moved in at once, so a crash midway
// never counts a session twice.
func migrate(dir string) error {
	legacy := dir + ".jsonl"
	raw, err := os.ReadFile(legacy)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	// Moved in already; only the removal did not happen.
	if _, err := os.Stat(dir); err == nil {
		return os.Remove(legacy)
	}

	days := map[string][]byte{}
	for _, line := range bytes.Split(raw, []byte("\n")) {
		var s Session
		if json.Unmarshal(line, &s) == nil {
			day := dayOf(s.EndedAt)
			days[day] = append(append(days[day], line...), '\n')
		}
	}

	tmp := dir + ".tmp"
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}
	if err := os.MkdirAll(tmp, 0o700); err != nil {
		return err
	}
	for day, text := range days {
		if err := os.WriteFile(filepath.Join(tmp, day+".jsonl"), text, 0o600); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, dir); err != nil {
		return err
	}
	return os.Remove(legacy)
}

func dayOf(t time.Time) string {
	return t.Local().Format(time.DateOnly)
}
