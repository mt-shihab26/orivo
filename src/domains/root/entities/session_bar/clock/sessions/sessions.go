package sessions

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
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

type History struct {
	path  string
	days  map[string]int
	stats map[string]Stat
}

func Open(path string) (*History, error) {
	h := &History{path: path, days: map[string]int{}, stats: map[string]Stat{}}

	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return h, nil
	}
	if err != nil {
		return h, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var s Session
		if json.Unmarshal(scanner.Bytes(), &s) == nil {
			h.count(s)
		}
	}
	return h, scanner.Err()
}

func (h *History) Record(s Session) error {
	h.count(s)
	s.StartedAt = s.StartedAt.Round(0)
	s.EndedAt = s.EndedAt.Round(0)

	line, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(h.path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(h.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(append(line, '\n'))
	return err
}

func (h *History) CountOn(day time.Time) int {
	return h.days[dayOf(day)]
}

func (h *History) Stat(todoID string) Stat {
	return h.stats[todoID]
}

func (h *History) count(s Session) {
	if s.Phase != "work" {
		return
	}
	h.days[dayOf(s.EndedAt)]++
	if s.TodoID != "" {
		stat := h.stats[s.TodoID]
		stat.Sessions++
		stat.Secs += s.DurationSecs
		h.stats[s.TodoID] = stat
	}
}

func dayOf(t time.Time) string {
	return t.Local().Format(time.DateOnly)
}
