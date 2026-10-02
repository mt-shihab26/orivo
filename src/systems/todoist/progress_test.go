package todoist

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProgressLineCountsSessionsAndTime(t *testing.T) {
	for _, tc := range []struct {
		sessions, secs int
		want           string
	}{
		{1, 25 * 60, "Worked on this for 25 minutes in 1 session."},
		{4, 100 * 60, "Worked on this for 1 hour and 40 minutes across 4 sessions."},
		{3, 2 * 3600, "Worked on this for 2 hours across 3 sessions."},
		{2, 61 * 60, "Worked on this for 1 hour and 1 minute across 2 sessions."},
		{1, 30, "Worked on this for less than a minute in 1 session."},
	} {
		if got := ProgressLine(tc.sessions, tc.secs); got != tc.want {
			t.Errorf("ProgressLine(%d, %d) = %q, want %q", tc.sessions, tc.secs, got, tc.want)
		}
	}
}

func TestMergeDescriptionKeepsTheUsersTextAndPutsTheLineLast(t *testing.T) {
	line := "Worked on this for 50 minutes across 2 sessions."
	old := "Worked on this for 25 minutes in 1 session."
	for _, tc := range []struct {
		name, description, want string
	}{
		{"empty", "", line},
		{"blank", " \n\n", line},
		{"user text", "Draft the intro.\nCite sources.", "Draft the intro.\nCite sources.\n\n" + line},
		{"replaces at the bottom", "Notes\n\n" + old, "Notes\n\n" + line},
		{"moves an old line to the bottom", old + "\nNotes", "Notes\n\n" + line},
		{"drops duplicates", "Notes\n" + old + "\nWorked on this for 4 hours across 9 sessions.", "Notes\n\n" + line},
		{"only orivo", old, line},
		{"keeps the user's own sentence", "Worked on this for ages.", "Worked on this for ages.\n\n" + line},
	} {
		if got := MergeDescription(tc.description, line); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// fakeTodoist answers the requests SetProgress makes. active and completed
// map task ids to descriptions; a write to an id in neither fails with 404.
type fakeTodoist struct {
	active, completed map[string]string
	down              bool
	readOnly          bool
	requests          int
}

func (f *fakeTodoist) client(t *testing.T) *Client {
	return serve(t, func(w http.ResponseWriter, r *http.Request) {
		f.requests++
		if f.down {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/tasks":
			var results []map[string]string
			for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
				if description, ok := f.active[id]; ok {
					results = append(results, map[string]string{"id": id, "description": description})
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"results": results, "next_cursor": nil})

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			description, ok := f.completed[strings.TrimPrefix(r.URL.Path, "/tasks/")]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"description": description, "checked": true})

		case r.Method == http.MethodPost && r.URL.Path == "/sync":
			if f.readOnly {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			var commands []command
			if err := json.Unmarshal([]byte(r.FormValue("commands")), &commands); err != nil {
				t.Error(err)
			}
			status := map[string]any{}
			for _, cmd := range commands {
				id := cmd.Args["id"]
				if cmd.Type != "item_update" {
					t.Errorf("command type = %q", cmd.Type)
				}
				_, active := f.active[id]
				_, completed := f.completed[id]
				switch {
				case active:
					f.active[id] = cmd.Args["description"]
				case completed:
					f.completed[id] = cmd.Args["description"]
				default:
					status[cmd.UUID] = map[string]any{"error": "Item not found", "http_code": 404}
					continue
				}
				status[cmd.UUID] = "ok"
			}
			json.NewEncoder(w).Encode(map[string]any{"sync_status": status})

		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
	})
}

func TestSetProgressWritesEveryTaskInOneBatch(t *testing.T) {
	todoist := &fakeTodoist{
		active: map[string]string{
			"a": "Notes\n\nWorked on this for 25 minutes in 1 session.",
			"b": "Draft",
			"c": "Draft",
		},
		completed: map[string]string{},
	}
	client := todoist.client(t)

	done, err := client.SetProgress(map[string]string{
		"a": "Worked on this for 50 minutes across 2 sessions.",
		"b": "Worked on this for 25 minutes in 1 session.",
		"c": "Worked on this for 25 minutes in 1 session.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(done) != 3 || todoist.requests != 2 {
		t.Fatalf("done = %v in %d requests, want all 3 in 2", done, todoist.requests)
	}
	if got := todoist.active["a"]; got != "Notes\n\nWorked on this for 50 minutes across 2 sessions." {
		t.Errorf("a = %q", got)
	}
	if got := todoist.active["b"]; got != "Draft\n\nWorked on this for 25 minutes in 1 session." {
		t.Errorf("b = %q", got)
	}
}

func TestSetProgressReachesCompletedTasksAndSettlesDeletedOnes(t *testing.T) {
	todoist := &fakeTodoist{
		active:    map[string]string{},
		completed: map[string]string{"done": "Finished it"},
	}

	settled, err := todoist.client(t).SetProgress(map[string]string{
		"done":    "Worked on this for 25 minutes in 1 session.",
		"deleted": "Worked on this for 25 minutes in 1 session.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(settled) != 2 {
		t.Fatalf("settled = %v, want both", settled)
	}
	if got := todoist.completed["done"]; got != "Finished it\n\nWorked on this for 25 minutes in 1 session." {
		t.Errorf("done = %q", got)
	}
}

func TestSetProgressSkipsTheWriteWhenNothingChanged(t *testing.T) {
	line := "Worked on this for 25 minutes in 1 session."
	todoist := &fakeTodoist{active: map[string]string{"a": "Notes\n\n" + line}}

	done, err := todoist.client(t).SetProgress(map[string]string{"a": line})
	if err != nil || len(done) != 1 || todoist.requests != 1 {
		t.Fatalf("done = %v, err = %v, requests = %d", done, err, todoist.requests)
	}
}

func TestSetProgressReportsAReadOnlyToken(t *testing.T) {
	todoist := &fakeTodoist{active: map[string]string{"a": ""}, readOnly: true}

	if _, err := todoist.client(t).SetProgress(map[string]string{"a": "Worked on this for 25 minutes in 1 session."}); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("err = %v, want ErrReadOnly", err)
	}
}

func TestOutboxKeepsWhatFailedForTheNextFlush(t *testing.T) {
	todoist := &fakeTodoist{active: map[string]string{"a": "Draft"}, completed: map[string]string{}, down: true}
	client := todoist.client(t)
	outbox := Outbox{Path: filepath.Join(t.TempDir(), "todoist-outbox.txt")}

	outbox.Add("a", "Worked on this for 25 minutes in 1 session.")
	outbox.Add("a", "Worked on this for 50 minutes across 2 sessions.")
	outbox.Add("gone", "Worked on this for 25 minutes in 1 session.")

	if err := outbox.Flush(client); err == nil {
		t.Fatal("flush while Todoist is down succeeded")
	}
	pending, _ := outbox.read()
	if len(pending) != 2 || pending["a"] != "Worked on this for 50 minutes across 2 sessions." {
		t.Fatalf("pending = %v, want both tasks with a's latest line", pending)
	}

	todoist.down = false
	if err := outbox.Flush(client); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outbox.Path); !os.IsNotExist(err) {
		t.Fatalf("outbox still exists after a full flush: %v", err)
	}
	if got := todoist.active["a"]; got != "Draft\n\nWorked on this for 50 minutes across 2 sessions." {
		t.Errorf("a = %q", got)
	}
}
