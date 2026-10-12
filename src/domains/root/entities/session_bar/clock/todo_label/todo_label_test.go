package todo_label

import (
	"errors"
	"image/color"
	"path/filepath"
	"testing"
	"time"

	"orivo/src/domains/root/core"
	"orivo/src/domains/root/entities/session_bar/clock/sessions"
	"orivo/src/systems/todoist"
	"orivo/src/systems/todos"
)

type fakeTimer struct{}

func (fakeTimer) Accent() color.RGBA        { return color.RGBA{} }
func (fakeTimer) TodoID() string            { return "" }
func (fakeTimer) TodoText() string          { return "" }
func (fakeTimer) Stat(string) sessions.Stat { return sessions.Stat{} }
func (fakeTimer) SetTodo(id, text string)   {}

func finishSync(t *testing.T, count int, err error) *TodoLabel {
	t.Helper()
	label := New(core.NewFonts("unused"), fakeTimer{}, "")

	label.syncing = true
	label.status = "Syncing Todoist…"
	label.results <- syncResult{count: count, err: err}
	label.Update(0)

	if label.syncing {
		t.Fatal("still marked as syncing after the result arrived")
	}
	return label
}

func TestSyncResultIsShownThenCleared(t *testing.T) {
	label := finishSync(t, 3, nil)
	if label.status != "Synced 3 todos from Todoist" {
		t.Fatalf("status = %q", label.status)
	}

	label.Update(statusSeconds - 1)
	if label.status == "" {
		t.Fatal("status cleared too early")
	}
	label.Update(2)
	if label.status != "" {
		t.Fatalf("status = %q, want it cleared", label.status)
	}
}

func TestSyncFailuresSayWhatToDo(t *testing.T) {
	cases := map[string]error{
		"Not connected to Todoist — run `orivo connect-todoist`": todoist.ErrNotConnected,
		"Todoist sign-in expired — run `orivo connect-todoist`":  todoist.ErrTokenRejected,
		"Sync failed: no network":                                errors.New("no network"),
	}
	for want, err := range cases {
		if got := finishSync(t, 0, err).status; got != want {
			t.Errorf("status = %q, want %q", got, want)
		}
	}
}

type pickedTimer struct {
	fakeTimer
	id string
}

func (p *pickedTimer) TodoID() string          { return p.id }
func (p *pickedTimer) SetTodo(id, text string) { p.id = id }

func TestSelectedTodoIsDroppedOnceSyncLeavesItOut(t *testing.T) {
	cache := todos.Cache{Path: filepath.Join(t.TempDir(), "todoist.txt")}
	timer := &pickedTimer{id: "2"}
	label := New(core.NewFonts("unused"), timer, cache.Path)

	label.Update(0)
	if timer.id != "2" {
		t.Fatalf("todo = %q, want it kept while there is no cache", timer.id)
	}

	due := time.Now()
	if err := cache.Write([]todos.Todo{{ID: "1", Text: "one", Due: due}, {ID: "2", Text: "two", Due: due}}); err != nil {
		t.Fatal(err)
	}
	label.Update(0)
	if timer.id != "2" {
		t.Fatalf("todo = %q, want it kept while it is still listed", timer.id)
	}

	if err := cache.Write([]todos.Todo{{ID: "1", Text: "one", Due: due}}); err != nil {
		t.Fatal(err)
	}
	label.Update(0)
	if timer.id != "" {
		t.Fatalf("todo = %q, want none after the sync left it out", timer.id)
	}
}

func TestTodoPickedFromAStaleListIsDropped(t *testing.T) {
	cache := todos.Cache{Path: filepath.Join(t.TempDir(), "todoist.txt")}
	if err := cache.Write([]todos.Todo{{ID: "1", Text: "one", Due: time.Now()}}); err != nil {
		t.Fatal(err)
	}
	timer := &pickedTimer{id: "1"}
	label := New(core.NewFonts("unused"), timer, cache.Path)
	label.Update(0)

	// The cache has not changed since, but the pick has.
	timer.id = "2"
	label.Update(0)
	if timer.id != "" {
		t.Fatalf("todo = %q, want none for one the cache does not list", timer.id)
	}
}
