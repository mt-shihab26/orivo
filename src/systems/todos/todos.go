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

var ErrNoCache = errors.New("no Todoist cache")

type Todo struct {
	ID   string
	Text string
	Due  time.Time
}

type Lists struct {
	Overdue []Todo
	Today   []Todo
}

type Source interface {
	Load(now time.Time) (Lists, error)
}

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
