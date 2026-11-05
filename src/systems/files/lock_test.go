package files

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLockWaitsForTheHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.txt")
	unlock, err := Lock(path)
	if err != nil {
		t.Fatal(err)
	}

	got := make(chan struct{})
	go func() {
		second, err := Lock(path)
		if err != nil {
			t.Error(err)
			return
		}
		second()
		close(got)
	}()

	select {
	case <-got:
		t.Fatal("a second lock was taken while the first was held")
	case <-time.After(50 * time.Millisecond):
	}
	unlock()
	select {
	case <-got:
	case <-time.After(time.Second):
		t.Fatal("the second lock was not taken once the first let go")
	}
}
