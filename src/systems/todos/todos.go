package todos

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"time"

	"orivo/src/systems/files"
)

var ErrNoCache = errors.New("no Todoist cache")

type Todo struct {
	ID   string
	Text string
	Due  time.Time
}

type Cache struct {
	Path string
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

	return files.WriteAtomic(c.Path, []byte(text.String()))
}

func oneLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func Split(all []Todo, now time.Time) (overdue, today []Todo) {
	y, m, d := now.Local().Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, time.Local)

	for _, todo := range all {
		switch {
		case todo.Due.Before(start):
			overdue = append(overdue, todo)
		case todo.Due.Equal(start):
			today = append(today, todo)
		}
	}

	slices.SortStableFunc(overdue, func(a, b Todo) int {
		return a.Due.Compare(b.Due)
	})
	return overdue, today
}
