package todoist

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"

	"orivo/src/systems/files"
)

// Completions holds the tasks marked done in orivo and not yet completed in
// Todoist, one id per line.
type Completions struct {
	Path string
}

func (c Completions) Add(taskID string) error {
	outboxMu.Lock()
	defer outboxMu.Unlock()

	pending, err := c.read()
	if err != nil {
		return err
	}
	if slices.Contains(pending, taskID) {
		return nil
	}
	return c.write(append(pending, taskID))
}

// Pending lists the tasks waiting to be completed in Todoist.
func (c Completions) Pending() ([]string, error) {
	outboxMu.Lock()
	defer outboxMu.Unlock()
	return c.read()
}

// Flush completes every pending task in Todoist in one batch and keeps those
// that failed.
func (c Completions) Flush(client *Client) error {
	outboxMu.Lock()
	pending, err := c.read()
	outboxMu.Unlock()
	if err != nil || len(pending) == 0 {
		return err
	}

	done, failed := client.Complete(pending)

	outboxMu.Lock()
	defer outboxMu.Unlock()

	// Tasks added while the batch was out stay.
	now, err := c.read()
	if err != nil {
		return err
	}
	now = slices.DeleteFunc(now, func(id string) bool { return slices.Contains(done, id) })
	if err := c.write(now); err != nil {
		return err
	}
	return failed
}

func (c Completions) read() ([]string, error) {
	raw, err := os.ReadFile(c.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(raw)), nil
}

func (c Completions) write(pending []string) error {
	if len(pending) == 0 {
		if err := os.Remove(c.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	return files.WriteAtomic(c.Path, []byte(strings.Join(pending, "\n")+"\n"))
}

// Complete closes each task in one request, as checking it off in Todoist
// does: a recurring task moves on to its next date. It returns the tasks that
// are settled: completed, or gone.
func (c *Client) Complete(ids []string) ([]string, error) {
	commands := make([]command, len(ids))
	for i, id := range ids {
		commands[i] = command{Type: "item_close", UUID: randomString(), Args: map[string]string{"id": id}}
	}

	statuses, err := c.sync(commands)
	if err != nil {
		return nil, err
	}
	var done []string
	var failed error
	for _, cmd := range commands {
		switch err := statuses[cmd.UUID]; {
		case err == nil, errors.Is(err, errTaskGone):
			done = append(done, cmd.Args["id"])
		default:
			failed = err
		}
	}
	return done, failed
}
