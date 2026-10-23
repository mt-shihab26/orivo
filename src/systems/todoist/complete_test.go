package todoist

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCompletionsKeepWhatFailedForTheNextFlush(t *testing.T) {
	todoist := &fakeTodoist{active: map[string]taskText{"a": {Content: "Draft"}}, completed: map[string]taskText{}, down: true}
	client := todoist.client(t)
	done := Completions{Path: filepath.Join(t.TempDir(), "todoist-done.txt")}

	done.Add("a")
	done.Add("a")
	done.Add("gone")

	if err := done.Flush(client); err == nil {
		t.Fatal("flush while Todoist is down succeeded")
	}
	if pending, _ := done.Pending(); !slices.Equal(pending, []string{"a", "gone"}) {
		t.Fatalf("pending = %q, want each task once", pending)
	}

	todoist.down = false
	if err := done.Flush(client); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(done.Path); !os.IsNotExist(err) {
		t.Fatalf("completions still exist after a full flush: %v", err)
	}
	if !slices.Equal(todoist.closed, []string{"a"}) {
		t.Errorf("closed = %q, want a", todoist.closed)
	}
}

func TestACompletedTaskKeepsItsLastTag(t *testing.T) {
	todoist := &fakeTodoist{active: map[string]taskText{"a": {Content: "Draft"}}, completed: map[string]taskText{}}
	client := todoist.client(t)
	dir := t.TempDir()
	outbox := Outbox{Path: filepath.Join(dir, "todoist-outbox.txt")}
	done := Completions{Path: filepath.Join(dir, "todoist-done.txt")}

	outbox.Add("a", "(1.4, 35 min)")
	done.Add("a")
	if err := outbox.Flush(client); err != nil {
		t.Fatal(err)
	}
	if err := done.Flush(client); err != nil {
		t.Fatal(err)
	}

	if got := todoist.active["a"].Content; got != "Draft (1.4, 35 min)" || !slices.Equal(todoist.closed, []string{"a"}) {
		t.Fatalf("a = %q, closed = %q", got, todoist.closed)
	}
}
