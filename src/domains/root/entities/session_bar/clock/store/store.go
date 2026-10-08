package store

import (
	"bytes"
	"encoding/json"
	"os"
	"time"

	"orivo/src/systems/files"
	"orivo/src/systems/logx"
)

const version = 2

const noTodo = "none"

type data struct {
	Version        int                  `json:"version"`
	TodoID         *string              `json:"timer_todo_id"`
	TodoText       string               `json:"timer_todo_text,omitempty"`
	Phase          string               `json:"timer_cycle_phase"`
	Remaining      map[string]int64     `json:"timer_remaining_millis"`
	PhaseStartedAt map[string]time.Time `json:"timer_phase_started_at"`
}

type Store struct {
	path string
	data data
	// What the file holds, so Save skips writing the same bytes again.
	saved []byte
}

func Load(path string) *Store {
	s := &Store{path: path, data: empty()}

	raw, err := os.ReadFile(path)
	if err != nil {
		return s
	}

	var probe struct {
		Version int `json:"version"`
	}
	if json.Unmarshal(raw, &probe) != nil {
		return s
	}
	if probe.Version != version {
		s.data = legacy(raw)
		return s
	}

	loaded := empty()
	if err := json.Unmarshal(raw, &loaded); err != nil {
		logx.Warn("store: ignoring unreadable %s: %v", path, err)
		return s
	}
	if loaded.Remaining == nil {
		loaded.Remaining = map[string]int64{}
	}
	if loaded.PhaseStartedAt == nil {
		loaded.PhaseStartedAt = map[string]time.Time{}
	}
	s.data = loaded
	return s
}

func empty() data {
	return data{
		Version:        version,
		Remaining:      map[string]int64{},
		PhaseStartedAt: map[string]time.Time{},
	}
}

func legacy(raw []byte) data {
	d := empty()

	var old struct {
		Phase     string           `json:"timer_cycle_phase"`
		Remaining map[string]int64 `json:"timer_remaining_millis"`
	}
	if json.Unmarshal(raw, &old) != nil {
		return d
	}
	d.Phase = old.Phase
	if millis, ok := old.Remaining[noTodo]; ok {
		d.Remaining[noTodo] = millis
	}
	return d
}

func (s *Store) Save() {
	raw, err := json.Marshal(s.data)
	if err != nil {
		logx.Error("store: encode failed: %v", err)
		return
	}
	if bytes.Equal(raw, s.saved) {
		return
	}
	if err := files.WriteAtomic(s.path, raw); err != nil {
		logx.Error("store: write failed: %v", err)
		return
	}
	s.saved = raw
}

func (s *Store) Todo() (id, text string) {
	if s.data.TodoID == nil {
		return "", ""
	}
	return *s.data.TodoID, s.data.TodoText
}

func (s *Store) SetTodo(id, text string) {
	if id == "" {
		s.data.TodoID = nil
		s.data.TodoText = ""
		return
	}
	s.data.TodoID = &id
	s.data.TodoText = text
}

func (s *Store) Phase() string {
	return s.data.Phase
}

func (s *Store) SetPhase(name string) {
	s.data.Phase = name
}

func (s *Store) Remaining(todoID string) (time.Duration, bool) {
	millis, ok := s.data.Remaining[key(todoID)]
	return time.Duration(millis) * time.Millisecond, ok
}

func (s *Store) SetRemaining(todoID string, d time.Duration) {
	s.data.Remaining[key(todoID)] = d.Milliseconds()
}

func (s *Store) ClearRemaining(todoID string) {
	delete(s.data.Remaining, key(todoID))
}

func (s *Store) ClearTimers() {
	clear(s.data.Remaining)
	clear(s.data.PhaseStartedAt)
}

func (s *Store) PhaseStartedAt(todoID string) (time.Time, bool) {
	t, ok := s.data.PhaseStartedAt[key(todoID)]
	return t, ok
}

func (s *Store) SetPhaseStartedAt(todoID string, t time.Time) {
	s.data.PhaseStartedAt[key(todoID)] = t.Round(0)
}

func (s *Store) ClearPhaseStartedAt(todoID string) {
	delete(s.data.PhaseStartedAt, key(todoID))
}

func key(todoID string) string {
	if todoID == "" {
		return noTodo
	}
	return todoID
}
