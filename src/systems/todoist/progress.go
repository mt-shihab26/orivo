package todoist

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"orivo/src/systems/files"
)

var (
	ErrReadOnly = errors.New("the Todoist sign-in can only read tasks")

	errTaskGone = errors.New("the Todoist task no longer exists")

	// Matches any sentence ProgressLine writes, so the old one is replaced.
	progressLine = regexp.MustCompile(`^Worked on this for .+ (in 1 session|across \d+ sessions)\.$`)
)

// ProgressLine is the sentence orivo keeps in a task's description.
func ProgressLine(sessions, secs int) string {
	if sessions == 1 {
		return fmt.Sprintf("Worked on this for %s in 1 session.", spent(secs))
	}
	return fmt.Sprintf("Worked on this for %s across %d sessions.", spent(secs), sessions)
}

func spent(secs int) string {
	h, m := secs/3600, secs%3600/60
	switch {
	case h > 0 && m > 0:
		return plural(h, "hour") + " and " + plural(m, "minute")
	case h > 0:
		return plural(h, "hour")
	case m > 0:
		return plural(m, "minute")
	}
	return "less than a minute"
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// MergeDescription drops orivo's old sentence from description and puts line
// at the bottom, below the user's text.
func MergeDescription(description, line string) string {
	var kept []string
	for _, l := range strings.Split(description, "\n") {
		if !progressLine.MatchString(strings.TrimSpace(l)) {
			kept = append(kept, l)
		}
	}

	text := strings.TrimRight(strings.Join(kept, "\n"), " \t\r\n")
	if text == "" {
		return line
	}
	return text + "\n\n" + line
}

func (c *Client) SetProgress(taskID, line string) error {
	path := "/tasks/" + url.PathEscape(taskID)

	var t struct {
		Description string `json:"description"`
	}
	if err := c.get(path, nil, &t); err != nil {
		return err
	}

	merged := MergeDescription(t.Description, line)
	if merged == t.Description {
		return nil
	}
	return c.post(path, map[string]string{"description": merged})
}

func (c *Client) post(path string, body any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return ErrTokenRejected
	// A token that reads fine but may not write was issued for data:read.
	case resp.StatusCode == http.StatusForbidden:
		return ErrReadOnly
	case resp.StatusCode == http.StatusNotFound:
		return errTaskGone
	case resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent:
		return fmt.Errorf("Todoist answered %s", resp.Status)
	}
	return nil
}

// Outbox holds the progress lines not yet written to Todoist, one per task,
// as the task id and the line separated by a tab.
type Outbox struct {
	Path string
}

// The app flushes from a goroutine per finished session and from its sync.
var outboxMu sync.Mutex

func (o Outbox) Add(taskID, line string) error {
	outboxMu.Lock()
	defer outboxMu.Unlock()

	pending, err := o.read()
	if err != nil {
		return err
	}
	pending[taskID] = line
	return o.write(pending)
}

// Flush writes every pending line to Todoist and keeps those that failed.
func (o Outbox) Flush(client *Client) error {
	outboxMu.Lock()
	defer outboxMu.Unlock()

	pending, err := o.read()
	if err != nil || len(pending) == 0 {
		return err
	}

	var failed error
	for taskID, line := range pending {
		err := client.SetProgress(taskID, line)
		if err == nil || errors.Is(err, errTaskGone) {
			delete(pending, taskID)
			continue
		}
		failed = err
		// The rest would fail the same way.
		if errors.Is(err, ErrTokenRejected) || errors.Is(err, ErrReadOnly) {
			break
		}
	}

	if err := o.write(pending); err != nil {
		return err
	}
	return failed
}

func (o Outbox) read() (map[string]string, error) {
	pending := map[string]string{}

	file, err := os.Open(o.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return pending, nil
	}
	if err != nil {
		return pending, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if id, line, ok := strings.Cut(scanner.Text(), "\t"); ok && id != "" {
			pending[id] = line
		}
	}
	return pending, scanner.Err()
}

func (o Outbox) write(pending map[string]string) error {
	if len(pending) == 0 {
		if err := os.Remove(o.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}

	var text strings.Builder
	for id, line := range pending {
		fmt.Fprintf(&text, "%s\t%s\n", id, line)
	}
	return files.WriteAtomic(o.Path, []byte(text.String()))
}

// PushProgress flushes the outbox at outboxPath with the saved sign-in.
func PushProgress(authPath, outboxPath string) error {
	token, err := NewOAuth().Token(authPath)
	if err != nil {
		return err
	}
	return Outbox{Path: outboxPath}.Flush(NewClient(token))
}
