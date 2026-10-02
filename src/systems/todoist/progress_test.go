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
		{1, 25 * 60, "orivo: 1 session · 25m"},
		{4, 100 * 60, "orivo: 4 sessions · 1h 40m"},
		{3, 2 * 3600, "orivo: 3 sessions · 2h 0m"},
	} {
		if got := ProgressLine(tc.sessions, tc.secs); got != tc.want {
			t.Errorf("ProgressLine(%d, %d) = %q, want %q", tc.sessions, tc.secs, got, tc.want)
		}
	}
}

func TestMergeDescriptionKeepsTheUsersTextAndPutsTheLineLast(t *testing.T) {
	line := "orivo: 2 sessions · 50m"
	for _, tc := range []struct {
		name, description, want string
	}{
		{"empty", "", line},
		{"blank", " \n\n", line},
		{"user text", "Draft the intro.\nCite sources.", "Draft the intro.\nCite sources.\n\n" + line},
		{"replaces at the bottom", "Notes\n\norivo: 1 session · 25m", "Notes\n\n" + line},
		{"moves an old line to the bottom", "orivo: 1 session · 25m\nNotes", "Notes\n\n" + line},
		{"drops duplicates", "Notes\norivo: 1 session · 25m\norivo: 9 sessions · 4h 0m", "Notes\n\n" + line},
		{"only orivo", "orivo: 1 session · 25m", line},
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
			w.Write([]byte(`{"id": "a", "description": "Notes\n\norivo: 1 session · 25m"}`))
			return
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		posted = body["description"]
		w.Write([]byte(`{"id": "a"}`))
	})

	if err := client.SetProgress("a", "orivo: 2 sessions · 50m"); err != nil {
		t.Fatal(err)
	}
	if posted != "Notes\n\norivo: 2 sessions · 50m" {
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

	if err := client.SetProgress("a", "orivo: 1 session · 25m"); !errors.Is(err, ErrReadOnly) {
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

	outbox.Add("a", "orivo: 1 session · 25m")
	outbox.Add("a", "orivo: 2 sessions · 50m")
	outbox.Add("gone", "orivo: 1 session · 25m")

	if err := outbox.Flush(client); err == nil {
		t.Fatal("flush offline succeeded")
	}
	pending, _ := outbox.read()
	if len(pending) != 1 || pending["a"] != "orivo: 2 sessions · 50m" {
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
