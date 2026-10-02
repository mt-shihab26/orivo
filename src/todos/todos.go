// Package todos supplies the todos the timer can be pointed at. orivo does
// not manage todos itself: they come from Todoist, and only the ones that
// are overdue or due today are offered.
package todos

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"
)

// ErrNoCache means no Todoist data has been cached yet.
var ErrNoCache = errors.New("no Todoist cache")

type Todo struct {
	ID   string
	Text string
	// Due is the local midnight of the day the todo is due.
	Due time.Time
}

// Lists are the pending todos the picker offers, in display order.
type Lists struct {
	Overdue []Todo
	Today   []Todo
}

// Source loads the pending todos that are overdue or due on now's local day.
type Source interface {
	Load(now time.Time) (Lists, error)
}

// FileCache reads Todoist tasks cached in a JSON file. It accepts the shapes
// Todoist itself uses: a bare array of tasks, or an object holding them under
// "items" (Sync API) or "results" (REST API).
type FileCache struct {
	Path string
}

func (c FileCache) Load(now time.Time) (Lists, error) {
	raw, err := os.ReadFile(c.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return Lists{}, ErrNoCache
	}
	if err != nil {
		return Lists{}, err
	}
	return Parse(raw, now)
}

type task struct {
	ID          json.RawMessage `json:"id"`
	Content     string          `json:"content"`
	Checked     bool            `json:"checked"`
	IsCompleted bool            `json:"is_completed"`
	IsDeleted   bool            `json:"is_deleted"`
	Due         *struct {
		Date string `json:"date"`
	} `json:"due"`
}

// Parse splits cached Todoist tasks into overdue and due-today lists.
func Parse(raw []byte, now time.Time) (Lists, error) {
	tasks, err := decode(raw)
	if err != nil {
		return Lists{}, err
	}

	y, m, d := now.Local().Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.Local)

	var lists Lists
	for _, t := range tasks {
		if t.Checked || t.IsCompleted || t.IsDeleted || t.Due == nil {
			continue
		}
		due, ok := dueDay(t.Due.Date)
		// Ids are strings in current Todoist data and numbers in older exports.
		id := strings.Trim(string(t.ID), `"`)
		if !ok || id == "" || id == "null" {
			continue
		}
		todo := Todo{ID: id, Text: t.Content, Due: due}
		switch {
		case due.Before(today):
			lists.Overdue = append(lists.Overdue, todo)
		case due.Equal(today):
			lists.Today = append(lists.Today, todo)
		}
	}

	sort.SliceStable(lists.Overdue, func(i, j int) bool {
		return lists.Overdue[i].Due.Before(lists.Overdue[j].Due)
	})
	return lists, nil
}

func decode(raw []byte) ([]task, error) {
	var tasks []task
	if bytes.HasPrefix(bytes.TrimSpace(raw), []byte("[")) {
		err := json.Unmarshal(raw, &tasks)
		return tasks, err
	}

	var wrapped struct {
		Items   []task `json:"items"`
		Results []task `json:"results"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, err
	}
	return append(wrapped.Items, wrapped.Results...), nil
}

// dueDay returns the local day of a Todoist due date, which is either a date
// (2026-10-02), a floating datetime (2026-10-02T15:00:00), or a UTC datetime
// (2026-10-02T15:00:00Z).
func dueDay(date string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339, date); err == nil {
		y, m, d := t.Local().Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.Local), true
	}
	if len(date) < len(time.DateOnly) {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(time.DateOnly, date[:len(time.DateOnly)], time.Local)
	return t, err == nil
}
