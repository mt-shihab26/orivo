package store

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"time"

	"orivo/src/systems/files"
	"orivo/src/systems/logx"
)

const version = 3

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
	if probe.Version == 2 {
		s.data = unphased(raw)
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

// unphased reads a version 2 store, whose timers were keyed by todo alone and
// all belonged to the phase it saved.
func unphased(raw []byte) data {
	d := empty()
	var old data
	if json.Unmarshal(raw, &old) != nil {
		return d
	}
	d.TodoID, d.TodoText, d.Phase = old.TodoID, old.TodoText, old.Phase
	for todoID, millis := range old.Remaining {
		d.Remaining[old.Phase+"/"+todoID] = millis
	}
	for todoID, t := range old.PhaseStartedAt {
		d.PhaseStartedAt[old.Phase+"/"+todoID] = t
	}
	return d
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
		d.Remaining[key(d.Phase, "")] = millis
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

func (s *Store) Remaining(phase, todoID string) (time.Duration, bool) {
	millis, ok := s.data.Remaining[key(phase, todoID)]
	return time.Duration(millis) * time.Millisecond, ok
}

func (s *Store) SetRemaining(phase, todoID string, d time.Duration) {
	s.data.Remaining[key(phase, todoID)] = d.Milliseconds()
}

func (s *Store) ClearRemaining(phase, todoID string) {
	delete(s.data.Remaining, key(phase, todoID))
}

// ClearPhase drops every todo's timer in the phase.
func (s *Store) ClearPhase(phase string) {
	prefix := phase + "/"
	for k := range s.data.Remaining {
		if strings.HasPrefix(k, prefix) {
			delete(s.data.Remaining, k)
		}
	}
	for k := range s.data.PhaseStartedAt {
		if strings.HasPrefix(k, prefix) {
			delete(s.data.PhaseStartedAt, k)
		}
	}
}

func (s *Store) PhaseStartedAt(phase, todoID string) (time.Time, bool) {
	t, ok := s.data.PhaseStartedAt[key(phase, todoID)]
	return t, ok
}

func (s *Store) SetPhaseStartedAt(phase, todoID string, t time.Time) {
	s.data.PhaseStartedAt[key(phase, todoID)] = t.Round(0)
}

func (s *Store) ClearPhaseStartedAt(phase, todoID string) {
	delete(s.data.PhaseStartedAt, key(phase, todoID))
}

// key names a todo's timer in one phase, so a todo paused in work keeps its
// time through the breaks other todos take in between.
func key(phase, todoID string) string {
	if todoID == "" {
		todoID = noTodo
	}
	return phase + "/" + todoID
}
