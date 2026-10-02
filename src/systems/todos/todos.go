package todos

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

type Cache struct {
	Path string
}

func (c Cache) Load(now time.Time) (Lists, error) {
	all, err := c.Read()
	if err != nil {
		return Lists{}, err
	}
	return Split(all, now), nil
}

func (c Cache) Read() ([]Todo, error) {
	file, err := os.Open(c.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNoCache
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var all []Todo
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), "\t", 3)
		if len(fields) != 3 {
			continue
		}
		due, err := time.ParseInLocation(time.DateOnly, fields[0], time.Local)
		if err != nil {
			continue
		}
		all = append(all, Todo{ID: fields[1], Text: fields[2], Due: due})
	}
	return all, scanner.Err()
}

func (c Cache) Write(all []Todo) error {
	var text strings.Builder
	for _, todo := range all {
		fmt.Fprintf(&text, "%s\t%s\t%s\n", todo.Due.Format(time.DateOnly), todo.ID, oneLine(todo.Text))
	}

	if err := os.MkdirAll(filepath.Dir(c.Path), 0o700); err != nil {
		return err
	}
	tmp := c.Path + ".tmp"
	if err := os.WriteFile(tmp, []byte(text.String()), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.Path)
}

func Split(all []Todo, now time.Time) Lists {
	y, m, d := now.Local().Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.Local)

	var lists Lists
	for _, todo := range all {
		switch {
		case todo.Due.Before(today):
			lists.Overdue = append(lists.Overdue, todo)
		case todo.Due.Equal(today):
			lists.Today = append(lists.Today, todo)
		}
	}

	sort.SliceStable(lists.Overdue, func(i, j int) bool {
		return lists.Overdue[i].Due.Before(lists.Overdue[j].Due)
	})
	return lists
}

func oneLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
