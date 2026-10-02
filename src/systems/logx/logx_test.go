package logx

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestWeeksStartOnSaturday(t *testing.T) {
	for _, tc := range []struct {
		day  string
		want string
	}{
		{"2026-09-26", "orivo-2026-09-26.log"}, // Saturday starts its own week
		{"2026-09-27", "orivo-2026-09-26.log"}, // Sunday
		{"2026-10-02", "orivo-2026-09-26.log"}, // Friday ends it
		{"2026-10-03", "orivo-2026-10-03.log"}, // the next Saturday starts a new one
		{"2027-01-01", "orivo-2026-12-26.log"}, // across a year
	} {
		day, err := time.ParseInLocation(time.DateOnly, tc.day, time.Local)
		if err != nil {
			t.Fatal(err)
		}
		if got := fileFor(day.Add(23 * time.Hour)); got != tc.want {
			t.Errorf("%s: got %s, want %s", tc.day, got, tc.want)
		}
	}
}

func TestPruneKeepsOnlyTheCurrentWeek(t *testing.T) {
	state := t.TempDir()
	dir := filepath.Join(state, "logs")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"logs/orivo-2026-09-19.log", "logs/orivo-2026-09-26.log", "orivo-2026-09-26.log", "orivo.log", "orivo.log.1", "store.json"} {
		if err := os.WriteFile(filepath.Join(state, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	prune(dir, "orivo-2026-09-26.log")

	var left []string
	filepath.WalkDir(state, func(path string, e os.DirEntry, err error) error {
		if err == nil && !e.IsDir() {
			rel, _ := filepath.Rel(state, path)
			left = append(left, rel)
		}
		return nil
	})
	if want := []string{"logs/orivo-2026-09-26.log", "store.json"}; !slices.Equal(left, want) {
		t.Fatalf("left = %v, want %v", left, want)
	}
}
