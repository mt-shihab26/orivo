package sessions

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type Session struct {
	Phase        string    `json:"phase"`
	DurationSecs int       `json:"duration_secs"`
	StartedAt    time.Time `json:"started_at"`
	EndedAt      time.Time `json:"ended_at"`
	TodoID       string    `json:"todo_id,omitempty"`
	TodoText     string    `json:"todo_text,omitempty"`
}

func Read(path string) ([]Session, error) {
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var all []Session
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var s Session
		if json.Unmarshal(scanner.Bytes(), &s) == nil {
			all = append(all, s)
		}
	}
	return all, scanner.Err()
}

func Append(path string, s Session) error {
	s.StartedAt = s.StartedAt.Round(0)
	s.EndedAt = s.EndedAt.Round(0)

	line, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(append(line, '\n'))
	return err
}
