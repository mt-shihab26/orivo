package todoist

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
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

func TestSetProgressRewritesTheDescription(t *testing.T) {
	posted := ""
	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/a" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		if r.Method == http.MethodGet {
			w.Write([]byte(`{"id": "a", "description": "Notes\n\nWorked on this for 25 minutes in 1 session."}`))
			return
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		posted = body["description"]
		w.Write([]byte(`{"id": "a"}`))
	})

	if err := client.SetProgress("a", "Worked on this for 50 minutes across 2 sessions."); err != nil {
		t.Fatal(err)
	}
	if posted != "Notes\n\nWorked on this for 50 minutes across 2 sessions." {
		t.Fatalf("posted %q", posted)
	}
}

func TestSetProgressReportsAReadOnlyToken(t *testing.T) {
	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Write([]byte(`{"id": "a", "description": ""}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
	})

	if err := client.SetProgress("a", "Worked on this for 25 minutes in 1 session."); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("err = %v, want ErrReadOnly", err)
	}
}

func TestOutboxKeepsWhatFailedForTheNextFlush(t *testing.T) {
	online := false
	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/tasks/gone":
			w.WriteHeader(http.StatusNotFound)
		case !online:
			w.WriteHeader(http.StatusServiceUnavailable)
		case r.Method == http.MethodGet:
			w.Write([]byte(`{"description": ""}`))
		default:
			w.Write([]byte(`{}`))
		}
	})
	outbox := Outbox{Path: filepath.Join(t.TempDir(), "todoist-outbox.txt")}

	outbox.Add("a", "Worked on this for 25 minutes in 1 session.")
	outbox.Add("a", "Worked on this for 50 minutes across 2 sessions.")
	outbox.Add("gone", "Worked on this for 25 minutes in 1 session.")

	if err := outbox.Flush(client); err == nil {
		t.Fatal("flush offline succeeded")
	}
	pending, _ := outbox.read()
	if len(pending) != 1 || pending["a"] != "Worked on this for 50 minutes across 2 sessions." {
		t.Fatalf("pending = %v, want only a's latest line", pending)
	}

	online = true
	if err := outbox.Flush(client); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outbox.Path); !os.IsNotExist(err) {
		t.Fatalf("outbox still exists after a full flush: %v", err)
	}
}
