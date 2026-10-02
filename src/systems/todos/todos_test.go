package todos

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func day(d int) time.Time {
	return time.Date(2026, 10, d, 0, 0, 0, 0, time.Local)
}

func TestCacheRoundTripsAsText(t *testing.T) {
	cache := Cache{Path: filepath.Join(t.TempDir(), "todoist.txt")}
	want := []Todo{
		{ID: "6X7rM8997g3RQmvh", Text: "Write the quarterly report", Due: day(2)},
		{ID: "6X7rM8997g3RQaaa", Text: "Reply to Ayşe — café booking", Due: day(1)},
	}

	if err := cache.Write(want); err != nil {
		t.Fatal(err)
	}
	got, err := cache.Read()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("read = %v, want %v", got, want)
	}
}

func TestCacheKeepsEachTodoOnOneLine(t *testing.T) {
	cache := Cache{Path: filepath.Join(t.TempDir(), "todoist.txt")}

	if err := cache.Write([]Todo{{ID: "1", Text: "first\tline\nsecond line", Due: day(2)}}); err != nil {
		t.Fatal(err)
	}
	got, err := cache.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Text != "first line second line" {
		t.Errorf("read = %v", got)
	}
}

func TestMissingCacheIsReportedAsSuch(t *testing.T) {
	_, err := Cache{Path: filepath.Join(t.TempDir(), "todoist.txt")}.Read()
	if !errors.Is(err, ErrNoCache) {
		t.Fatalf("err = %v, want ErrNoCache", err)
	}
}
