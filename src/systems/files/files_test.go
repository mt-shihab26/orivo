package files

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrentWritesLeaveOneWholeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todoist.txt")

	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := range 20 {
		wg.Go(func() {
			errs <- WriteAtomic(path, fmt.Appendf(nil, "write %02d\n", i))
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil || len(raw) != len("write 00\n") {
		t.Fatalf("file = %q, err = %v", raw, err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("left behind %d files, want only todoist.txt", len(entries))
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
}
