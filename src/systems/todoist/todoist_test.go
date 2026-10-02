package todoist

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func serve(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := NewClient("secret")
	client.BaseURL = server.URL
	return client
}

func TestDueTodosFollowsPagesAndMapsTasks(t *testing.T) {
	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/filter" || r.URL.Query().Get("query") != "today | overdue" {
			t.Errorf("unexpected request %s", r.URL)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("authorization = %q", got)
		}

		if r.URL.Query().Get("cursor") == "" {
			w.Write([]byte(`{"next_cursor": "page2", "results": [
				{"id": "a", "content": "dated",   "due": {"date": "2026-10-02"}},
				{"id": "b", "content": "timed",   "due": {"date": "2026-10-01T15:00:00"}},
				{"id": "c", "content": "done",    "due": {"date": "2026-10-02"}, "checked": true},
				{"id": "d", "content": "no date", "due": null}
			]}`))
			return
		}
		w.Write([]byte(`{"next_cursor": null, "results": [
			{"id": "e", "content": "second page", "due": {"date": "2026-09-30"}}
		]}`))
	})

	got, err := client.DueTodos()
	if err != nil {
		t.Fatal(err)
	}

	want := []struct {
		id, text string
		day      int
		month    time.Month
	}{
		{"a", "dated", 2, time.October},
		{"b", "timed", 1, time.October},
		{"e", "second page", 30, time.September},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d todos, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].ID != w.id || got[i].Text != w.text || got[i].Due.Day() != w.day || got[i].Due.Month() != w.month {
			t.Errorf("todo %d = %+v, want %+v", i, got[i], w)
		}
	}
}

func TestRejectedTokenIsReportedAsSuch(t *testing.T) {
	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	if _, err := client.User(); !errors.Is(err, ErrTokenRejected) {
		t.Fatalf("err = %v, want ErrTokenRejected", err)
	}
}

func TestUserReturnsTheAccountName(t *testing.T) {
	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("unexpected request %s", r.URL)
		}
		w.Write([]byte(`{"full_name": "Shihab", "email": "s@example.com"}`))
	})

	name, err := client.User()
	if err != nil || name != "Shihab" {
		t.Fatalf("name = %q, err = %v", name, err)
	}
}

func TestTokenIsSavedAndLoaded(t *testing.T) {
	t.Setenv(tokenEnv, "")
	path := filepath.Join(t.TempDir(), "todoist.token")

	if _, err := LoadToken(path); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("err = %v, want ErrNotConnected", err)
	}
	if err := SaveToken(path, "abc123"); err != nil {
		t.Fatal(err)
	}
	if token, err := LoadToken(path); err != nil || token != "abc123" {
		t.Fatalf("token = %q, err = %v", token, err)
	}

	t.Setenv(tokenEnv, "from-env")
	if token, _ := LoadToken(path); token != "from-env" {
		t.Fatalf("token = %q, want the environment's", token)
	}
}
