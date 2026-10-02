package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mt-shihab26/orivo/src/phase"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	started := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)

	s := Load(path)
	s.SetTodo("abc", "Write report")
	s.SetPhase(phase.LongBreak)
	s.SetRemaining("abc", 90*time.Second)
	s.SetPhaseStartedAt("abc", started)
	s.Save()

	s = Load(path)
	if id, text := s.Todo(); id != "abc" || text != "Write report" {
		t.Errorf("todo = %q %q", id, text)
	}
	if s.Phase() != phase.LongBreak {
		t.Errorf("phase = %v", s.Phase())
	}
	if d, ok := s.Remaining("abc"); !ok || d != 90*time.Second {
		t.Errorf("remaining = %v %v", d, ok)
	}
	if got, ok := s.PhaseStartedAt("abc"); !ok || !got.Equal(started) {
		t.Errorf("phase started at = %v %v", got, ok)
	}
}

// A store left behind by the terminal app refers to todos from its own
// database; only the phase and the no-todo time carry over.
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
	if s.Phase() != phase.Break {
		t.Errorf("phase = %v, want Break", s.Phase())
	}
	if d, ok := s.Remaining(""); !ok || d != 2*time.Minute {
		t.Errorf("no-todo remaining = %v %v, want 2m", d, ok)
	}
	if _, ok := s.Remaining("49"); ok {
		t.Error("remaining for old todo 49 was kept")
	}
}
