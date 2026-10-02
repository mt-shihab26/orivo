// Package sessions records completed timer sessions and answers the few
// questions the timer asks about them.
package sessions

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const workPhase = "work"

type Session struct {
	Phase        string    `json:"phase"`
	DurationSecs int       `json:"duration_secs"`
	StartedAt    time.Time `json:"started_at"`
	EndedAt      time.Time `json:"ended_at"`
	TodoID       string    `json:"todo_id,omitempty"`
	// TodoText is a snapshot, so history stays readable after the todo is
	// completed or deleted in Todoist.
	TodoText string `json:"todo_text,omitempty"`
}

// Stat aggregates the completed work sessions of one todo.
type Stat struct {
	Sessions int
	Secs     int
}

// Log is an append-only JSON-lines file of sessions, indexed in memory.
type Log struct {
	mu    sync.Mutex
	path  string
	stats map[string]Stat
	// Completed work sessions per local day (YYYY-MM-DD).
	days map[string]int
}

// Open reads the log at path; a missing file is an empty log.
func Open(path string) (*Log, error) {
	l := &Log{path: path, stats: map[string]Stat{}, days: map[string]int{}}

	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return l, nil
	}
	if err != nil {
		return l, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var s Session
		// A torn or hand-edited line should not cost the rest of the history.
		if json.Unmarshal(scanner.Bytes(), &s) == nil {
			l.index(s)
		}
	}
	return l, scanner.Err()
}

// Record appends a session to the log.
func (l *Log) Record(s Session) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	s.StartedAt = s.StartedAt.Round(0)
	s.EndedAt = s.EndedAt.Round(0)
	l.index(s)

	line, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(append(line, '\n'))
	return err
}

// CountToday returns the number of work sessions that ended on now's local day.
func (l *Log) CountToday(now time.Time) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.days[day(now)]
}

// Stat returns the work session totals for the given todo.
func (l *Log) Stat(todoID string) Stat {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stats[todoID]
}

func (l *Log) index(s Session) {
	if s.Phase != workPhase {
		return
	}
	l.days[day(s.EndedAt)]++
	if s.TodoID != "" {
		stat := l.stats[s.TodoID]
		stat.Sessions++
		stat.Secs += s.DurationSecs
		l.stats[s.TodoID] = stat
	}
}

func day(t time.Time) string {
	return t.Local().Format(time.DateOnly)
}
