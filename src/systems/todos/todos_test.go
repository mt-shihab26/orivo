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

func texts(list []Todo) []string {
	out := []string{}
	for _, todo := range list {
		out = append(out, todo.Text)
	}
	return out
}

func TestSplitsTodosIntoOverdueAndToday(t *testing.T) {
	now := time.Date(2026, 10, 2, 14, 30, 0, 0, time.Local)

	overdue, today := Split([]Todo{
		{ID: "1", Text: "today", Due: day(2)},
		{ID: "2", Text: "yesterday", Due: day(1)},
		{ID: "3", Text: "last week", Due: time.Date(2026, 9, 25, 0, 0, 0, 0, time.Local)},
		{ID: "4", Text: "tomorrow", Due: day(3)},
		{ID: "5", Text: "also today", Due: day(2)},
	}, now)

	if got, want := texts(overdue), []string{"last week", "yesterday"}; !slices.Equal(got, want) {
		t.Errorf("overdue = %v, want %v", got, want)
	}
	if got, want := texts(today), []string{"today", "also today"}; !slices.Equal(got, want) {
		t.Errorf("today = %v, want %v", got, want)
	}

	overdue, today = Split([]Todo{{ID: "1", Text: "was due today", Due: day(2)}}, now.AddDate(0, 0, 1))
	if len(overdue) != 1 || len(today) != 0 {
		t.Errorf("a day later: overdue = %v, today = %v", overdue, today)
	}
}
