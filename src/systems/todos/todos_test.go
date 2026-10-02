package todos

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

var now = time.Date(2026, 10, 2, 14, 30, 0, 0, time.Local)

func texts(list []Todo) []string {
	out := []string{}
	for _, todo := range list {
		out = append(out, todo.Text)
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSplitsPendingTodosIntoOverdueAndToday(t *testing.T) {
	raw := []byte(`[
		{"id": "1", "content": "today",          "due": {"date": "2026-10-02"}},
		{"id": "2", "content": "today at 3pm",   "due": {"date": "2026-10-02T15:00:00"}},
		{"id": "3", "content": "yesterday",      "due": {"date": "2026-10-01"}},
		{"id": "4", "content": "last week",      "due": {"date": "2026-09-25"}},
		{"id": "5", "content": "tomorrow",       "due": {"date": "2026-10-03"}},
		{"id": "6", "content": "no date",        "due": null},
		{"id": "7", "content": "done",           "due": {"date": "2026-10-02"}, "checked": true},
		{"id": "8", "content": "deleted",        "due": {"date": "2026-10-02"}, "is_deleted": true},
		{"id": 9,   "content": "numeric id",     "due": {"date": "2026-10-02"}}
	]`)

	lists, err := Parse(raw, now)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := texts(lists.Overdue), []string{"last week", "yesterday"}; !equal(got, want) {
		t.Errorf("overdue = %v, want %v", got, want)
	}
	if got, want := texts(lists.Today), []string{"today", "today at 3pm", "numeric id"}; !equal(got, want) {
		t.Errorf("today = %v, want %v", got, want)
	}
	if got := lists.Today[2].ID; got != "9" {
		t.Errorf("numeric id = %q, want 9", got)
	}
}

func TestAcceptsTodoistResponseWrappers(t *testing.T) {
	for _, raw := range []string{
		`{"items":   [{"id": "1", "content": "a", "due": {"date": "2026-10-02"}}]}`,
		`{"results": [{"id": "1", "content": "a", "due": {"date": "2026-10-02"}}]}`,
	} {
		lists, err := Parse([]byte(raw), now)
		if err != nil || len(lists.Today) != 1 {
			t.Errorf("%s: today = %v, err = %v", raw, lists.Today, err)
		}
	}
}

func TestMissingCacheIsReportedAsSuch(t *testing.T) {
	_, err := FileCache{Path: filepath.Join(t.TempDir(), "todoist.json")}.Load(now)
	if !errors.Is(err, ErrNoCache) {
		t.Fatalf("err = %v, want ErrNoCache", err)
	}
}
