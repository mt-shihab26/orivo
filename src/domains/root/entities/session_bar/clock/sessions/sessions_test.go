package sessions

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func at(d, h int) time.Time {
	return time.Date(2026, 10, d, h, 0, 0, 0, time.Local)
}

func work(todoID string, ended time.Time) Session {
	return Session{Phase: "work", DurationSecs: 25 * 60, StartedAt: ended.Add(-25 * time.Minute), EndedAt: ended, TodoID: todoID}
}

func TestEachDayHasItsOwnFileAndCount(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	h, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range []Session{work("a", at(2, 9)), work("a", at(2, 10)), work("a", at(3, 9))} {
		if err := h.Record(s); err != nil {
			t.Fatal(err)
		}
	}

	if got := h.StatOn(at(2, 12), "a"); got.Sessions != 2 || got.Secs != 50*60 {
		t.Errorf("Oct 2 = %+v, want 2 sessions", got)
	}
	if got := h.StatOn(at(3, 12), "a"); got.Sessions != 1 {
		t.Errorf("Oct 3 = %+v, want 1 session", got)
	}
	if got := h.CountOn(at(4, 12)); got != 0 {
		t.Errorf("Oct 4 count = %v, want 0", got)
	}
	for _, name := range []string{"2026-10-02.jsonl", "2026-10-03.jsonl"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
}

func TestOpenSplitsTheOldSingleFileByDay(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	old := `{"phase":"work","duration_secs":1500,"ended_at":"2026-10-02T09:00:00Z","todo_id":"a"}
{"phase":"work","duration_secs":1500,"ended_at":"2026-10-03T09:00:00Z","todo_id":"a"}
not json
`
	if err := os.WriteFile(dir+".jsonl", []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}

	h, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(dir + ".jsonl"); !os.IsNotExist(err) {
		t.Errorf("old file still there: %v", err)
	}
	day := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
	if got := h.StatOn(day, "a"); got.Sessions != 1 {
		t.Errorf("Oct 2 = %+v, want 1 session", got)
	}

	// Opening again must not split anything twice.
	h, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := h.StatOn(day, "a"); got.Sessions != 1 {
		t.Errorf("reopened Oct 2 = %+v, want 1 session", got)
	}
}

func TestUndoRemovesOnlyTheLastSession(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	h, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := h.Undo(at(2, 12)); ok || err != nil {
		t.Fatalf("undo on an empty day: ok=%v err=%v", ok, err)
	}

	for _, s := range []Session{work("a", at(2, 9)), work("b", at(2, 10))} {
		if err := h.Record(s); err != nil {
			t.Fatal(err)
		}
	}
	// A line cut short by a crash is skipped, and lends nothing to the one before.
	file := filepath.Join(dir, "2026-10-02.jsonl")
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, append(raw, []byte(`{"phase":"break","todo_id":"zzz","duration_secs":"x"}`+"\n")...), 0o600); err != nil {
		t.Fatal(err)
	}

	last, ok, err := h.Undo(at(2, 12))
	if !ok || err != nil || last.TodoID != "b" || last.Phase != "work" {
		t.Fatalf("undo = %+v ok=%v err=%v, want the session on b", last, ok, err)
	}
	if h.CountOn(at(2, 12)) != 1 || h.StatOn(at(2, 12), "a").Sessions != 1 || h.StatOn(at(2, 12), "b").Sessions != 0 {
		t.Fatalf("count = %v, want only the session on a left", h.CountOn(at(2, 12)))
	}
}

func TestSharesAddUpButEachEndedSessionCountsOnce(t *testing.T) {
	h, err := Open(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	short := work("a", at(2, 10))
	short.DurationSecs, short.Share = 10*60, 0.4
	for _, s := range []Session{work("a", at(2, 9)), short} {
		if err := h.Record(s); err != nil {
			t.Fatal(err)
		}
	}

	// Read back from the file, not from what Record counted.
	h = &History{dir: h.dir}
	if got := h.CountOn(at(2, 12)); got != 1.4 {
		t.Errorf("count = %v, want 1.4", got)
	}
	if got := h.EndedOn(at(2, 12)); got != 2 {
		t.Errorf("ended = %v, want 2", got)
	}
	if got := h.StatOn(at(2, 12), "a"); got.Sessions != 1.4 || got.Secs != 35*60 {
		t.Errorf("stat = %+v, want 1.4 sessions of 35 min", got)
	}
}

func TestFormatShowsOneDecimalAtMost(t *testing.T) {
	for count, want := range map[float64]string{0: "0", 3: "3", 2.4: "2.4", 0.36: "0.4", 2.96: "3"} {
		if got := Format(count); got != want {
			t.Errorf("Format(%v) = %q, want %q", count, got, want)
		}
	}
}
