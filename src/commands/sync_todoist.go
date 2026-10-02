package commands

import (
	"errors"
	"fmt"
	"time"

	"github.com/mt-shihab26/orivo/src/entities/sessionbar/clock/todolabel/todopicker"
	"github.com/mt-shihab26/orivo/src/systems/paths"
	"github.com/mt-shihab26/orivo/src/systems/todoist"
	"github.com/mt-shihab26/orivo/src/systems/todos"
)

type SyncTodoist struct{}

func (c *SyncTodoist) Name() string {
	return "sync-todoist"
}

func (c *SyncTodoist) Summary() string {
	return "Fetch todos that are overdue or due today, and cache them"
}

func (c *SyncTodoist) Run(args []string) error {
	token, err := todoist.LoadToken(paths.TodoistToken())
	if errors.Is(err, todoist.ErrNotConnected) {
		return errors.New("not connected to Todoist; run `orivo connect-todoist` first")
	}
	if err != nil {
		return err
	}

	all, err := todoist.NewClient(token).DueTodos()
	if errors.Is(err, todoist.ErrTokenRejected) {
		return errors.New("Todoist rejected the saved token; run `orivo connect-todoist` again")
	}
	if err != nil {
		return fmt.Errorf("could not fetch todos: %w", err)
	}

	if err := (todos.Cache{Path: paths.TodoistCache()}).Write(all); err != nil {
		return err
	}

	overdue, today := todopicker.Split(all, time.Now())
	printTodos("Overdue", overdue, true)
	printTodos("Today", today, false)

	fmt.Printf("Cached %d todos in %s\n", len(overdue)+len(today), paths.TodoistCache())
	return nil
}

func printTodos(title string, list []todos.Todo, withDue bool) {
	if len(list) == 0 {
		return
	}
	fmt.Println(title)
	for _, todo := range list {
		if withDue {
			fmt.Printf("  %-6s  %s\n", todo.Due.Format("Jan 2"), todo.Text)
		} else {
			fmt.Printf("  %s\n", todo.Text)
		}
	}
	fmt.Println()
}
