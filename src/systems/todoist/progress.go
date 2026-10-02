package todoist

import (
	"bufio"

	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"

	"orivo/src/systems/files"
)

var (
	ErrReadOnly = errors.New("the Todoist sign-in can only read tasks")

	errTaskGone = errors.New("the Todoist task no longer exists")

	// Matches the tags ProgressTag writes at the end of a title, and the
	// wordier ones it wrote before, so the old ones are replaced.
	progressTag = regexp.MustCompile(`(\s*\(\d+( sessions?)?, \d+ min\))+$`)

	// Matches the sentence older versions kept in the description, so it is
	// cleared out.
	progressLine = regexp.MustCompile(`^Worked on this for .+ (in 1 session|across \d+ sessions)\.$`)
)

// ProgressTag is what orivo keeps at the end of a task's title. Recurring
// tasks lose their description when completed, but keep their title.
func ProgressTag(sessions, secs int) string {
	return fmt.Sprintf("(%d, %d min)", sessions, secs/60)
}

// StripTitle drops orivo's tag from title, so the counts Todoist holds are
// never read back.
func StripTitle(title string) string {
	return strings.TrimRight(progressTag.ReplaceAllString(title, ""), " \t")
}

// MergeTitle replaces orivo's old tag at the end of title with tag.
func MergeTitle(title, tag string) string {
	if text := StripTitle(title); text != "" {
		return text + " " + tag
	}
	return tag
}

// cleanDescription drops the sentence older versions wrote to description.
func cleanDescription(description string) string {
	var kept []string
	for _, l := range strings.Split(description, "\n") {
		if !progressLine.MatchString(strings.TrimSpace(l)) {
			kept = append(kept, l)
		}
	}
	if len(kept) == len(strings.Split(description, "\n")) {
		return description
	}
	return strings.TrimRight(strings.Join(kept, "\n"), " \t\r\n")
}

type taskText struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	Description string `json:"description"`
}

// SetProgress puts each task's tag in its title, with one request to read
// them and one to write them however many tasks there are. It returns the
// tasks that are settled: written, already up to date, or gone.
func (c *Client) SetProgress(tags map[string]string) ([]string, error) {
	ids := slices.Sorted(maps.Keys(tags))
	found, err := c.texts(ids)
	if err != nil {
		return nil, err
	}

	var done []string
	var commands []command
	for _, id := range ids {
		t, ok := found[id]
		if !ok {
			// Not active any more; a task completed since still takes the tag.
			err := c.get("/tasks/"+url.PathEscape(id), nil, &t)
			if errors.Is(err, errTaskGone) {
				done = append(done, id)
				continue
			}
			if err != nil {
				return done, err
			}
		}

		args := map[string]string{"id": id}
		if title := MergeTitle(t.Content, tags[id]); title != t.Content {
			args["content"] = title
		}
		if description := cleanDescription(t.Description); description != t.Description {
			args["description"] = description
		}
		if len(args) == 1 {
			done = append(done, id)
			continue
		}
		commands = append(commands, command{Type: "item_update", UUID: randomString(), Args: args})
	}
	if len(commands) == 0 {
		return done, nil
	}

	statuses, err := c.sync(commands)
	if err != nil {
		return done, err
	}
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

// texts fetches the titles and descriptions of the active tasks among ids.
func (c *Client) texts(ids []string) (map[string]taskText, error) {
	found := map[string]taskText{}
	cursor := ""

	for range maxPages {
		query := url.Values{"ids": {strings.Join(ids, ",")}, "limit": {pageSize}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}

		var page struct {
			Results    []taskText `json:"results"`
			NextCursor string     `json:"next_cursor"`
		}
		if err := c.get("/tasks", query, &page); err != nil {
			return nil, err
		}
		for _, t := range page.Results {
			found[t.ID] = t
		}

		cursor = page.NextCursor
		if cursor == "" {
			return found, nil
		}
	}
	return nil, errors.New("Todoist kept returning more pages than expected")
}

type command struct {
	Type string            `json:"type"`
	UUID string            `json:"uuid"`
	Args map[string]string `json:"args"`
}

// sync sends commands in one request and returns each one's error by uuid.
func (c *Client) sync(commands []command) (map[string]error, error) {
	raw, err := json.Marshal(commands)
	if err != nil {
		return nil, err
	}

	form := url.Values{"commands": {string(raw)}}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/sync", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, ErrTokenRejected
	// A token that reads fine but may not write was issued for data:read.
	case resp.StatusCode == http.StatusForbidden:
		return nil, ErrReadOnly
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("Todoist answered %s", resp.Status)
	}

	// Each status is the string "ok" or an object describing the error.
	var body struct {
		SyncStatus map[string]json.RawMessage `json:"sync_status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	statuses := map[string]error{}
	for _, cmd := range commands {
		status, ok := body.SyncStatus[cmd.UUID]
		if !ok {
			statuses[cmd.UUID] = errors.New("Todoist did not answer the update")
			continue
		}
		if string(status) == `"ok"` {
			statuses[cmd.UUID] = nil
			continue
		}

		var failure struct {
			Error    string `json:"error"`
			HTTPCode int    `json:"http_code"`
		}
		json.Unmarshal(status, &failure)
		switch failure.HTTPCode {
		case http.StatusNotFound:
			statuses[cmd.UUID] = errTaskGone
		case http.StatusForbidden:
			statuses[cmd.UUID] = ErrReadOnly
		default:
			statuses[cmd.UUID] = fmt.Errorf("Todoist refused the update: %s", failure.Error)
		}
	}
	return statuses, nil
}

// Outbox holds the progress tags not yet written to Todoist, one per task,
// as the task id and the tag separated by a tab.
type Outbox struct {
	Path string
}

// Guards the file only; it is never held over a request, so Add from the UI
// thread does not wait on a flush in the background.
var outboxMu sync.Mutex

func (o Outbox) Add(taskID, tag string) error {
	outboxMu.Lock()
	defer outboxMu.Unlock()

	pending, err := o.read()
	if err != nil {
		return err
	}
	pending[taskID] = tag
	return o.write(pending)
}

// Flush writes every pending tag to Todoist in one batch and keeps those
// that failed.
func (o Outbox) Flush(client *Client) error {
	outboxMu.Lock()
	pending, err := o.read()
	outboxMu.Unlock()
	if err != nil || len(pending) == 0 {
		return err
	}

	done, failed := client.SetProgress(pending)

	outboxMu.Lock()
	defer outboxMu.Unlock()

	// Tags added while the batch was out are newer, so they stay.
	now, err := o.read()
	if err != nil {
		return err
	}
	for _, id := range done {
		if now[id] == pending[id] {
			delete(now, id)
		}
	}
	if err := o.write(now); err != nil {
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
		// Sentences older versions queued for the description are dropped.
		if id, tag, ok := strings.Cut(scanner.Text(), "\t"); ok && id != "" && progressTag.MatchString(tag) {
			pending[id] = tag
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
	for id, tag := range pending {
		fmt.Fprintf(&text, "%s\t%s\n", id, tag)
	}
	return files.WriteAtomic(o.Path, []byte(text.String()))
}

// PushProgress flushes the outbox at outboxPath with the saved sign-in.
func PushProgress(authPath, outboxPath string) error {
	outbox := Outbox{Path: outboxPath}
	// Nothing queued means no request at all, not even a token refresh.
	if _, err := os.Stat(outboxPath); errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	token, err := NewOAuth().Token(authPath)
	if err != nil {
		return err
	}
	return outbox.Flush(NewClient(token))
}
