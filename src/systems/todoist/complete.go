package todoist

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"

	"orivo/src/systems/files"
)

// reopenMark starts the line of a task to reopen, one done in orivo whose
// completion was undone after it had reached Todoist.
const reopenMark = "-"

// Completions holds the tasks marked done in orivo and not yet completed in
// Todoist, one id per line, and those to reopen, each id after a "-".
type Completions struct {
	Path string
}

func (c Completions) Add(taskID string) error {
	return c.queue(taskID, reopenMark+taskID)
}

// Reopen takes back the task's completion: it only drops it while it is
// still waiting here, and reopens the task in Todoist once it was sent.
func (c Completions) Reopen(taskID string) error {
	return c.queue(reopenMark+taskID, taskID)
}

// queue adds line, unless its opposite is still waiting, in which case the
// two cancel out and Todoist never hears of either.
func (c Completions) queue(line, opposite string) error {
	unlock, err := files.Lock(c.Path)
	if err != nil {
		return err
	}
	defer unlock()

	pending, err := c.read()
	if err != nil {
		return err
	}
	if i := slices.Index(pending, opposite); i >= 0 {
		return c.write(slices.Delete(pending, i, i+1))
	}
	if slices.Contains(pending, line) {
		return nil
	}
	return c.write(append(pending, line))
}

// Pending lists the tasks waiting to be completed in Todoist. Writes replace
// the file whole, so a read needs no lock.
func (c Completions) Pending() ([]string, error) {
	lines, err := c.read()
	return slices.DeleteFunc(lines, isReopen), err
}

// Reopening lists the tasks waiting to be reopened in Todoist.
func (c Completions) Reopening() ([]string, error) {
	lines, err := c.read()
	var ids []string
	for _, line := range lines {
		if isReopen(line) {
			ids = append(ids, strings.TrimPrefix(line, reopenMark))
		}
	}
	return ids, err
}

func isReopen(line string) bool {
	return strings.HasPrefix(line, reopenMark)
}

// Flush completes and reopens every pending task in Todoist in one batch and
// keeps those that failed.
func (c Completions) Flush(client *Client) error {
	pending, err := c.read()
	if err != nil || len(pending) == 0 {
		return err
	}

	done, failed := client.Complete(pending)

	unlock, err := files.Lock(c.Path)
	if err != nil {
		return err
	}
	defer unlock()

	// Lines added while the batch was out stay. One sent but cancelled out
	// meanwhile already reached Todoist, so its opposite is sent next time.
	now, err := c.read()
	if err != nil {
		return err
	}
	for _, line := range done {
		if i := slices.Index(now, line); i >= 0 {
			now = slices.Delete(now, i, i+1)
		} else {
			now = append(now, opposite(line))
		}
	}
	if err := c.write(now); err != nil {
		return err
	}
	return failed
}

func opposite(line string) string {
	if isReopen(line) {
		return strings.TrimPrefix(line, reopenMark)
	}
	return reopenMark + line
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
// does: a recurring task moves on to its next date. A line marked to reopen
// uncompletes its task instead, which leaves a recurring one on the date it
// moved on to. It returns the lines that are settled: done, or gone.
func (c *Client) Complete(lines []string) ([]string, error) {
	commands := make([]command, len(lines))
	for i, line := range lines {
		kind, id := "item_close", line
		if isReopen(line) {
			kind, id = "item_uncomplete", strings.TrimPrefix(line, reopenMark)
		}
		commands[i] = command{Type: kind, UUID: randomString(), Args: map[string]string{"id": id}}
	}

	statuses, err := c.sync(commands)
	if err != nil {
		return nil, err
	}
	var done []string
	var failed error
	for i, cmd := range commands {
		switch err := statuses[cmd.UUID]; {
		case err == nil, errors.Is(err, errTaskGone):
			done = append(done, lines[i])
		default:
			failed = err
		}
	}
	return done, failed
}
