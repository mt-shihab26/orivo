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

func TestReopenBeforeTheSendCancelsTheCompletion(t *testing.T) {
	todoist := &fakeTodoist{active: map[string]taskText{"a": {Content: "Draft"}}}
	client := todoist.client(t)
	done := Completions{Path: filepath.Join(t.TempDir(), "todoist-done.txt")}

	done.Add("a")
	done.Reopen("a")
	if err := done.Flush(client); err != nil {
		t.Fatal(err)
	}
	if todoist.requests != 0 {
		t.Fatalf("requests = %d, want none for a completion taken back", todoist.requests)
	}
	if _, err := os.Stat(done.Path); !os.IsNotExist(err) {
		t.Fatalf("completions still exist: %v", err)
	}
}

func TestReopenAfterTheSendReopensTheTask(t *testing.T) {
	todoist := &fakeTodoist{active: map[string]taskText{"a": {Content: "Draft"}}}
	client := todoist.client(t)
	done := Completions{Path: filepath.Join(t.TempDir(), "todoist-done.txt")}

	done.Add("a")
	if err := done.Flush(client); err != nil {
		t.Fatal(err)
	}
	done.Reopen("a")
	if pending, _ := done.Pending(); len(pending) != 0 {
		t.Fatalf("pending = %q, want a reopening only", pending)
	}
	if reopening, _ := done.Reopening(); !slices.Equal(reopening, []string{"a"}) {
		t.Fatalf("reopening = %q, want a", reopening)
	}
	if err := done.Flush(client); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(todoist.closed, []string{"a"}) || !slices.Equal(todoist.reopened, []string{"a"}) {
		t.Fatalf("closed = %q, reopened = %q, want a closed then reopened", todoist.closed, todoist.reopened)
	}
	if _, err := os.Stat(done.Path); !os.IsNotExist(err) {
		t.Fatalf("completions still exist: %v", err)
	}
}

func TestReopenWhileTheSendIsOutReopensAfterIt(t *testing.T) {
	todoist := &fakeTodoist{active: map[string]taskText{"a": {Content: "Draft"}}}
	client := todoist.client(t)
	done := Completions{Path: filepath.Join(t.TempDir(), "todoist-done.txt")}

	done.Add("a")
	todoist.duringSync = func() {
		todoist.duringSync = nil
		done.Reopen("a")
	}
	if err := done.Flush(client); err != nil {
		t.Fatal(err)
	}
	if reopening, _ := done.Reopening(); !slices.Equal(reopening, []string{"a"}) {
		t.Fatalf("reopening = %q, want a, as its completion had already gone out", reopening)
	}
}

func TestDoneAgainBeforeTheReopenIsSentCancelsIt(t *testing.T) {
	done := Completions{Path: filepath.Join(t.TempDir(), "todoist-done.txt")}
	if err := os.WriteFile(done.Path, []byte("-a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	done.Add("a")
	if _, err := os.Stat(done.Path); !os.IsNotExist(err) {
		t.Fatalf("completions still exist: %v", err)
	}
}
