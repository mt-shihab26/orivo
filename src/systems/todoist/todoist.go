package todoist

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"orivo/src/systems/todos"
)

const (
	defaultBaseURL = "https://api.todoist.com/api/v1"
	dueQuery       = "today | overdue"
	pageSize       = "200"
	maxPages       = 50
	tokenEnv       = "TODOIST_API_TOKEN"
)

var (
	ErrNotConnected  = errors.New("not connected to Todoist")
	ErrTokenRejected = errors.New("Todoist rejected the token")
)

type Client struct {
	Token   string
	BaseURL string
	HTTP    *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		Token:   token,
		BaseURL: defaultBaseURL,
		HTTP:    &http.Client{Timeout: 20 * time.Second},
	}
}

type task struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Checked bool   `json:"checked"`
	Due     *struct {
		Date string `json:"date"`
	} `json:"due"`
}

func (c *Client) User() (string, error) {
	var user struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
	}
	if err := c.get("/user", nil, &user); err != nil {
		return "", err
	}
	if user.FullName != "" {
		return user.FullName, nil
	}
	return user.Email, nil
}

func (c *Client) DueTodos() ([]todos.Todo, error) {
	var all []todos.Todo
	cursor := ""

	for range maxPages {
		query := url.Values{"query": {dueQuery}, "limit": {pageSize}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}

		var page struct {
			Results    []task `json:"results"`
			NextCursor string `json:"next_cursor"`
		}
		if err := c.get("/tasks/filter", query, &page); err != nil {
			return nil, err
		}

		for _, t := range page.Results {
			if t.Checked || t.Due == nil || t.ID == "" {
				continue
			}
			if due, ok := dueDay(t.Due.Date); ok {
				all = append(all, todos.Todo{ID: t.ID, Text: t.Content, Due: due})
			}
		}

		cursor = page.NextCursor
		if cursor == "" {
			return all, nil
		}
	}
	return nil, errors.New("Todoist kept returning more pages than expected")
}

func (c *Client) get(path string, query url.Values, out any) error {
	target := c.BaseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return ErrTokenRejected
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("Todoist answered %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
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

func LoadToken(path string) (string, error) {
	if token := strings.TrimSpace(os.Getenv(tokenEnv)); token != "" {
		return token, nil
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrNotConnected
	}
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return "", ErrNotConnected
	}
	return token, nil
}

func SaveToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(token+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
