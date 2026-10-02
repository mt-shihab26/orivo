package todoist

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	logRequest = func(string, ...any) {}
	os.Exit(m.Run())
}

func TestEveryRequestIsLogged(t *testing.T) {
	var lines []string
	logRequest = func(format string, args ...any) { lines = append(lines, fmt.Sprintf(format, args...)) }
	t.Cleanup(func() { logRequest = func(string, ...any) {} })

	client := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results": [], "next_cursor": null}`))
	})
	client.HTTP = newHTTP()

	client.texts([]string{"a", "b"})
	client.texts([]string{"c"})

	if len(lines) != 2 {
		t.Fatalf("logged %d lines, want 2: %q", len(lines), lines)
	}
	if !strings.Contains(lines[0], "GET ") || !strings.Contains(lines[0], "/tasks?ids=a,b") || !strings.Contains(lines[0], "200 OK") {
		t.Errorf("line = %q", lines[0])
	}
	for _, line := range lines {
		if strings.Contains(line, "secret") {
			t.Errorf("token leaked into the log: %q", line)
		}
	}
}
