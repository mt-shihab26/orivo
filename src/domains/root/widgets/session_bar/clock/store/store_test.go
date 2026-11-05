package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	started := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)

	s := Load(path)
	s.SetTodo("abc", "Write report")
	s.SetPhase("LongBreak")
	s.SetRemaining("LongBreak", "abc", 90*time.Second)
	s.SetPhaseStartedAt("LongBreak", "abc", started)
	s.Save()

	s = Load(path)
	if id, text := s.Todo(); id != "abc" || text != "Write report" {
		t.Errorf("todo = %q %q", id, text)
	}
	if s.Phase() != "LongBreak" {
		t.Errorf("phase = %v", s.Phase())
	}
	if d, ok := s.Remaining("LongBreak", "abc"); !ok || d != 90*time.Second {
		t.Errorf("remaining = %v %v", d, ok)
	}
	if got, ok := s.PhaseStartedAt("LongBreak", "abc"); !ok || !got.Equal(started) {
		t.Errorf("phase started at = %v %v", got, ok)
	}
}

func TestLegacyStoreKeepsPhaseAndDropsOldTodos(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	old := `{"timer_todo_id":49,"timer_cycle_phase":"Break",
		"timer_remaining_millis":{"none":120000,"49":3300000},
		"timer_phase_started_at":{"49":[2026,256,12,40,59,150196977,6,0,0]}}`
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}

	s := Load(path)
	if id, _ := s.Todo(); id != "" {
		t.Errorf("todo = %q, want none", id)
	}
	if s.Phase() != "Break" {
		t.Errorf("phase = %v, want Break", s.Phase())
	}
	if d, ok := s.Remaining("Break", ""); !ok || d != 2*time.Minute {
		t.Errorf("no-todo remaining = %v %v, want 2m", d, ok)
	}
	if _, ok := s.Remaining("Break", "49"); ok {
		t.Error("remaining for old todo 49 was kept")
	}
}

func TestSaveSkipsTheWriteWhenNothingChanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	s := Load(path)
	s.SetPhase("work")
	s.Save()

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	s.Save()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unchanged store was written again: %v", err)
	}

	s.SetPhase("break")
	s.Save()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("changed store was not written: %v", err)
	}
}

func TestVersion2TimersMoveIntoTheirPhase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	old := `{"version":2,"timer_todo_id":"abc","timer_todo_text":"Write report","timer_cycle_phase":"Work",
		"timer_remaining_millis":{"abc":600000,"none":90000},"timer_phase_started_at":{}}`
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}

	s := Load(path)
	if id, text := s.Todo(); id != "abc" || text != "Write report" {
		t.Errorf("todo = %q %q", id, text)
	}
	if d, ok := s.Remaining("Work", "abc"); !ok || d != 10*time.Minute {
		t.Errorf("work remaining = %v %v, want 10m", d, ok)
	}
	if d, ok := s.Remaining("Work", ""); !ok || d != 90*time.Second {
		t.Errorf("no-todo remaining = %v %v, want 90s", d, ok)
	}
	if _, ok := s.Remaining("Break", "abc"); ok {
		t.Error("work time leaked into the break")
	}
}

func TestClearPhaseKeepsOtherPhases(t *testing.T) {
	s := Load(filepath.Join(t.TempDir(), "store.json"))
	s.SetRemaining("Work", "a", time.Minute)
	s.SetRemaining("Break", "a", time.Minute)
	s.SetRemaining("Break", "b", time.Minute)

	s.ClearPhase("Break")
	if _, ok := s.Remaining("Work", "a"); !ok {
		t.Error("work time was cleared")
	}
	if _, ok := s.Remaining("Break", "a"); ok {
		t.Error("break time for a was kept")
	}
	if _, ok := s.Remaining("Break", "b"); ok {
		t.Error("break time for b was kept")
	}
}
