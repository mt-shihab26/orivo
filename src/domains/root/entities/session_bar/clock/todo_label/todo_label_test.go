package todo_label

import (
	"errors"
	"image/color"
	"testing"

	"orivo/src/domains/root/core"
	"orivo/src/domains/root/entities/session_bar/clock/sessions"
	"orivo/src/systems/todoist"
)

type fakeTimer struct{}

func (fakeTimer) Accent() color.RGBA        { return color.RGBA{} }
func (fakeTimer) TodoID() string            { return "" }
func (fakeTimer) TodoText() string          { return "" }
func (fakeTimer) Stat(string) sessions.Stat { return sessions.Stat{} }
func (fakeTimer) SetTodo(id, text string)   {}

func finishSync(t *testing.T, count int, err error) *TodoLabel {
	t.Helper()
	label := New(core.NewFonts("unused"), fakeTimer{})

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
