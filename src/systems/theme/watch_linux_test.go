package theme

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// switchTheme does what omarchy-theme-set does: replace the theme folder,
// then write theme.name.
func switchTheme(t *testing.T, dir, name, colors string) {
	t.Helper()
	writeTheme(t, dir, "next-theme", colors)
	if err := os.RemoveAll(filepath.Join(dir, "theme")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "next-theme"), filepath.Join(dir, "theme")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "theme.name"), []byte(name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func next(t *testing.T, w *Watcher) Theme {
	t.Helper()
	select {
	case got := <-w.Themes:
		return got
	case <-time.After(2 * time.Second):
		t.Fatal("no theme after the switch")
		return Theme{}
	}
}

func TestWatchFollowsThemeSwitches(t *testing.T) {
	dir := t.TempDir()
	switchTheme(t, dir, "tokyo-night", tokyoNight)

	w, err := Watch(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	// Twice, since a watch on colors.toml itself would stop after the first.
	switchTheme(t, dir, "red", `background = "#ff0000"`)
	if got := next(t, w); got.Background != (color.RGBA{0xff, 0, 0, 255}) {
		t.Fatalf("background = %v after the first switch", got.Background)
	}

	switchTheme(t, dir, "blue", `background = "#0000ff"`)
	if got := next(t, w); got.Background != (color.RGBA{0, 0, 0xff, 255}) {
		t.Fatalf("background = %v after the second switch", got.Background)
	}
}

func TestWatchIgnoresOtherFiles(t *testing.T) {
	dir := t.TempDir()
	switchTheme(t, dir, "tokyo-night", tokyoNight)

	w, err := Watch(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	if err := os.WriteFile(filepath.Join(dir, "other"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-w.Themes:
		t.Fatalf("reloaded on an unrelated file: %+v", got)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestWatchFailsWithoutTheFolder(t *testing.T) {
	if _, err := Watch(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("watching a missing folder succeeded")
	}
}

func TestCloseStopsTheWatcher(t *testing.T) {
	w, err := Watch(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}
