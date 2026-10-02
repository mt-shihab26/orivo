package sync_todoist

import (
	"errors"
	"fmt"
	"time"

	"orivo/src/systems/paths"
	"orivo/src/systems/todoist"
	"orivo/src/systems/todos"
)

func Run() error {
	all, err := todoist.Sync(paths.TodoistAuth(), paths.TodoistCache())
	if errors.Is(err, todoist.ErrNotConnected) {
		return errors.New("not connected to Todoist; run `orivo connect-todoist` first")
	}
	if errors.Is(err, todoist.ErrTokenRejected) {
		return errors.New("the Todoist sign-in has expired; run `orivo connect-todoist` again")
	}
	if err != nil {
		return fmt.Errorf("could not sync with Todoist: %w", err)
	}

	overdue, today := todos.Split(all, time.Now())
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
