package todo_picker

import (
	"slices"
	"testing"
	"time"

	"orivo/src/systems/todos"
)

func texts(list []todos.Todo) []string {
	out := []string{}
	for _, todo := range list {
		out = append(out, todo.Text)
	}
	return out
}

func TestSplitsTodosIntoOverdueAndToday(t *testing.T) {
	day := func(month time.Month, d int) time.Time {
		return time.Date(2026, month, d, 0, 0, 0, 0, time.Local)
	}
	now := time.Date(2026, 10, 2, 14, 30, 0, 0, time.Local)

	overdue, today := Split([]todos.Todo{
		{ID: "1", Text: "today", Due: day(10, 2)},
		{ID: "2", Text: "yesterday", Due: day(10, 1)},
		{ID: "3", Text: "last week", Due: day(9, 25)},
		{ID: "4", Text: "tomorrow", Due: day(10, 3)},
		{ID: "5", Text: "also today", Due: day(10, 2)},
	}, now)

	if got, want := texts(overdue), []string{"last week", "yesterday"}; !slices.Equal(got, want) {
		t.Errorf("overdue = %v, want %v", got, want)
	}
	if got, want := texts(today), []string{"today", "also today"}; !slices.Equal(got, want) {
		t.Errorf("today = %v, want %v", got, want)
	}

	overdue, today = Split([]todos.Todo{{ID: "1", Text: "was due today", Due: day(10, 2)}}, now.AddDate(0, 0, 1))
	if len(overdue) != 1 || len(today) != 0 {
		t.Errorf("a day later: overdue = %v, today = %v", overdue, today)
	}
}
