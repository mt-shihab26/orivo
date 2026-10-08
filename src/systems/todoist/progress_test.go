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

func TestProgressTagCountsSessionsAndMinutes(t *testing.T) {
	for _, tc := range []struct {
		sessions, secs int
		want           string
	}{
		{1, 25 * 60, "(1, 25 min)"},
		{4, 100 * 60, "(4, 100 min)"},
		{1, 30, "(1, 0 min)"},
	} {
		if got := ProgressTag(tc.sessions, tc.secs); got != tc.want {
			t.Errorf("ProgressTag(%d, %d) = %q, want %q", tc.sessions, tc.secs, got, tc.want)
		}
	}
}

func TestMergeTitleReplacesTheOldTag(t *testing.T) {
	tag := "(2, 50 min)"
	for _, tc := range []struct {
		name, title, want string
	}{
		{"empty", "", tag},
		{"plain", "Write report", "Write report " + tag},
		{"replaces", "Write report (1, 25 min)", "Write report " + tag},
		{"drops duplicates", "Write report (1, 25 min) (9, 240 min)", "Write report " + tag},
		{"replaces the wordier tag", "Write report (1 session, 25 min)", "Write report " + tag},
		{"keeps the user's own brackets", "Call (Ayşe)", "Call (Ayşe) " + tag},
		{"only orivo", "(1, 25 min)", tag},
	} {
		if got := MergeTitle(tc.title, tag); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestCleanDescriptionDropsTheOldSentence(t *testing.T) {
	for _, tc := range []struct {
		description, want string
	}{
		{"", ""},
		{"Notes", "Notes"},
		{"Notes\n\nWorked on this for 25 minutes in 1 session.", "Notes"},
		{"Worked on this for 1 hour across 2 sessions.", ""},
		{"Worked on this for ages.", "Worked on this for ages."},
	} {
		if got := cleanDescription(tc.description); got != tc.want {
			t.Errorf("cleanDescription(%q) = %q, want %q", tc.description, got, tc.want)
		}
	}
}

// fakeTodoist answers the requests SetProgress makes. active and completed
// map task ids to their text; a write to an id in neither fails with 404.
type fakeTodoist struct {
	active, completed map[string]taskText
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
			var results []taskText
			for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
				if task, ok := f.active[id]; ok {
					task.ID = id
					results = append(results, task)
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"results": results, "next_cursor": nil})

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			task, ok := f.completed[strings.TrimPrefix(r.URL.Path, "/tasks/")]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(task)

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
				if cmd.Type != "item_update" {
					t.Errorf("command type = %q", cmd.Type)
				}
				tasks := f.active
				if _, ok := tasks[cmd.Args["id"]]; !ok {
					tasks = f.completed
				}
				task, ok := tasks[cmd.Args["id"]]
				if !ok {
					status[cmd.UUID] = map[string]any{"error": "Item not found", "http_code": 404}
					continue
				}
				if content, ok := cmd.Args["content"]; ok {
					task.Content = content
				}
				if description, ok := cmd.Args["description"]; ok {
					task.Description = description
				}
				tasks[cmd.Args["id"]] = task
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
		active: map[string]taskText{
			"a": {Content: "Report (1, 25 min)", Description: "Notes\n\nWorked on this for 25 minutes in 1 session."},
			"b": {Content: "Draft", Description: "Keep me"},
			"c": {Content: "Review"},
		},
		completed: map[string]taskText{},
	}
	client := todoist.client(t)

	done, err := client.SetProgress(map[string]string{
		"a": "(2, 50 min)",
		"b": "(1, 25 min)",
		"c": "(1, 25 min)",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(done) != 3 || todoist.requests != 2 {
		t.Fatalf("done = %v in %d requests, want all 3 in 2", done, todoist.requests)
	}
	if got := todoist.active["a"]; got != (taskText{Content: "Report (2, 50 min)", Description: "Notes"}) {
		t.Errorf("a = %+v", got)
	}
	if got := todoist.active["b"]; got != (taskText{Content: "Draft (1, 25 min)", Description: "Keep me"}) {
		t.Errorf("b = %+v", got)
	}
}

func TestSetProgressReachesCompletedTasksAndSettlesDeletedOnes(t *testing.T) {
	todoist := &fakeTodoist{
		active:    map[string]taskText{},
		completed: map[string]taskText{"done": {Content: "Finished it"}},
	}

	settled, err := todoist.client(t).SetProgress(map[string]string{
		"done":    "(1, 25 min)",
		"deleted": "(1, 25 min)",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(settled) != 2 {
		t.Fatalf("settled = %v, want both", settled)
	}
	if got := todoist.completed["done"].Content; got != "Finished it (1, 25 min)" {
		t.Errorf("done = %q", got)
	}
}

func TestSetProgressSkipsTheWriteWhenNothingChanged(t *testing.T) {
	todoist := &fakeTodoist{active: map[string]taskText{"a": {Content: "Report (1, 25 min)", Description: "Notes"}}}

	done, err := todoist.client(t).SetProgress(map[string]string{"a": "(1, 25 min)"})
	if err != nil || len(done) != 1 || todoist.requests != 1 {
		t.Fatalf("done = %v, err = %v, requests = %d", done, err, todoist.requests)
	}
}

func TestSetProgressReportsAReadOnlyToken(t *testing.T) {
	todoist := &fakeTodoist{active: map[string]taskText{"a": {Content: "Report"}}, readOnly: true}

	if _, err := todoist.client(t).SetProgress(map[string]string{"a": "(1, 25 min)"}); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("err = %v, want ErrReadOnly", err)
	}
}

func TestOutboxKeepsWhatFailedForTheNextFlush(t *testing.T) {
	todoist := &fakeTodoist{active: map[string]taskText{"a": {Content: "Draft"}}, completed: map[string]taskText{}, down: true}
	client := todoist.client(t)
	outbox := Outbox{Path: filepath.Join(t.TempDir(), "todoist-outbox.txt")}

	outbox.Add("a", "(1, 25 min)")
	outbox.Add("a", "(2, 50 min)")
	outbox.Add("gone", "(1, 25 min)")

	if err := outbox.Flush(client); err == nil {
		t.Fatal("flush while Todoist is down succeeded")
	}
	pending, _ := outbox.read()
	if len(pending) != 2 || pending["a"] != "(2, 50 min)" {
		t.Fatalf("pending = %v, want both tasks with a's latest tag", pending)
	}

	todoist.down = false
	if err := outbox.Flush(client); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outbox.Path); !os.IsNotExist(err) {
		t.Fatalf("outbox still exists after a full flush: %v", err)
	}
	if got := todoist.active["a"].Content; got != "Draft (2, 50 min)" {
		t.Errorf("a = %q", got)
	}
}

func TestOutboxDropsSentencesOlderVersionsQueued(t *testing.T) {
	outbox := Outbox{Path: filepath.Join(t.TempDir(), "todoist-outbox.txt")}
	old := "a\tWorked on this for 25 minutes in 1 session.\nb\t(1, 25 min)\n"
	if err := os.WriteFile(outbox.Path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}

	pending, err := outbox.read()
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending["b"] != "(1, 25 min)" {
		t.Fatalf("pending = %v, want only b's tag", pending)
	}
}

func TestMergeTitleWithoutATagOnlyDropsTheOldOne(t *testing.T) {
	for _, tc := range []struct {
		name, title, want string
	}{
		{"drops", "Write report (1, 25 min)", "Write report"},
		{"untagged", "Write report", "Write report"},
		{"keeps a bare tag as the title", "(1, 25 min)", "(1, 25 min)"},
	} {
		if got := MergeTitle(tc.title, ""); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestOutboxClearDropsOldTagsButNotNewerOnes(t *testing.T) {
	todoist := &fakeTodoist{active: map[string]taskText{
		"a": {Content: "Draft (2, 50 min)"},
		"b": {Content: "Review (3, 75 min)"},
	}, completed: map[string]taskText{}}
	outbox := Outbox{Path: filepath.Join(t.TempDir(), "todoist-outbox.txt")}

	// A session on b ended while the sync was out.
	outbox.Add("b", "(1, 25 min)")
	if err := outbox.Clear([]string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if err := outbox.Flush(todoist.client(t)); err != nil {
		t.Fatal(err)
	}

	if got := todoist.active["a"].Content; got != "Draft" {
		t.Errorf("a = %q, want the tag gone", got)
	}
	if got := todoist.active["b"].Content; got != "Review (1, 25 min)" {
		t.Errorf("b = %q, want today's tag", got)
	}
	if _, err := os.Stat(outbox.Path); !os.IsNotExist(err) {
		t.Fatalf("outbox still exists after a full flush: %v", err)
	}
}
