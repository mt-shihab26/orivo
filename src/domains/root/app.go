package root

import (
	"os"
	"os/signal"
	"slices"
	"sync/atomic"
	"syscall"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/domains/root/core"
	"orivo/src/domains/root/entities/hints"
	"orivo/src/domains/root/entities/session_bar"
	"orivo/src/domains/root/entities/top_bar"
	"orivo/src/systems/config"
	"orivo/src/systems/logx"
	"orivo/src/systems/theme"
)

const (
	frame     = time.Second / 60
	heartbeat = time.Second
)

type App struct {
	fonts    *core.Fonts
	entities []core.Entity
	quit     atomic.Bool
	themes   *theme.Watcher
	dirty    bool

	// Frames drawn in the last full second, and so far in this one.
	fps, draws int
	second     time.Time
}

func New(cfg config.Config) *App {
	rl.SetTraceLogLevel(rl.LogWarning)
	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagMsaa4xHint)
	rl.InitWindow(core.BaseWidth, core.BaseHeight, "Orivo")
	rl.SetWindowMinSize(420, 360)
	rl.SetTargetFPS(60)
	rl.SetExitKey(0)
	rl.InitAudioDevice()

	a := &App{
		fonts: core.NewFonts(cfg.Font),
	}
	a.loadTheme()

	a.entities = []core.Entity{
		top_bar.New(a.fonts, cfg.ShowFPS, a.FPS, a.Quit),
		hints.New(a.fonts),
		session_bar.New(cfg.Timer, a.fonts),
	}

	onInterrupt(a.Quit)
	return a
}

// loadTheme uses the current Omarchy theme, falling back to orivo's own
// colors, and follows theme switches while orivo runs. Without Omarchy it
// changes nothing.
func (a *App) loadTheme() {
	dir := theme.OmarchyDir()

	t, err := theme.Load(dir)
	if err != nil {
		logx.Warn("failed to read the Omarchy theme: %v", err)
	}
	core.ApplyTheme(t)

	if _, err := os.Stat(dir); err != nil {
		return
	}
	a.themes, err = theme.Watch(dir)
	if err != nil {
		logx.Warn("failed to watch the Omarchy theme: %v", err)
	}
}

func (a *App) Close() {
	if a.themes != nil {
		a.themes.Close()
	}
	for _, entity := range slices.Backward(a.entities) {
		entity.Close()
	}
	a.fonts.Close()
	rl.CloseAudioDevice()
	rl.CloseWindow()
}

func (a *App) Quit() {
	a.quit.Store(true)
}

// Run updates 60 times a second but draws only when something on screen may
// have changed: on input, when an entity says it changed, and at least once
// a second for anything else. A paused timer then costs a fraction of the
// CPU it took to draw every frame.
func (a *App) Run() {
	last := time.Now()
	var drawn time.Time
	for !rl.WindowShouldClose() && !a.quit.Load() {
		now := time.Now()
		a.Update(float32(now.Sub(last).Seconds()))
		last = now
		if now.Sub(a.second) >= time.Second {
			a.fps, a.draws, a.second = a.draws, 0, now
		}

		if a.dirty || input() || a.changed() || now.Sub(drawn) >= heartbeat {
			// EndDrawing polls the input and waits out the rest of the frame.
			a.Draw()
			a.dirty = false
			drawn = now
			a.draws++
			continue
		}
		time.Sleep(frame)
		rl.PollInputEvents()
	}
}

func (a *App) changed() bool {
	for _, entity := range a.entities {
		if c, ok := entity.(core.Changer); ok && c.Changed() {
			return true
		}
	}
	return false
}

// input reports whether a key, mouse or window event came in since the last
// poll. Keys held down count too, so repeats redraw.
func input() bool {
	if rl.IsWindowResized() || rl.GetKeyPressed() != 0 || rl.GetMouseWheelMove() != 0 {
		return true
	}
	if delta := rl.GetMouseDelta(); delta.X != 0 || delta.Y != 0 {
		return true
	}
	for button := rl.MouseButtonLeft; button <= rl.MouseButtonBack; button++ {
		if rl.IsMouseButtonDown(button) || rl.IsMouseButtonReleased(button) {
			return true
		}
	}
	for key := int32(rl.KeySpace); key <= rl.KeyKbMenu; key++ {
		if rl.IsKeyDown(key) || rl.IsKeyReleased(key) {
			return true
		}
	}
	return false
}

func (a *App) Update(dt float32) {
	if a.themes != nil {
		select {
		case t := <-a.themes.Themes:
			core.ApplyTheme(t)
			a.dirty = true
		case err := <-a.themes.Errors:
			logx.Warn("failed to read the Omarchy theme: %v", err)
		default:
		}
	}

	for _, entity := range a.entities {
		entity.Update(dt)
	}
}

func (a *App) Draw() {
	a.fonts.Ensure(core.CurrentScreen().Scale)

	rl.BeginDrawing()
	defer rl.EndDrawing()
	rl.ClearBackground(core.ColorBackground)

	for _, entity := range a.entities {
		entity.Draw()
	}
}

func onInterrupt(fn func()) {
	received := make(chan os.Signal, 1)
	signal.Notify(received, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-received
		fn()
	}()
}

// FPS is how many frames were drawn in the last second. raylib's own count
// times only the frames that are drawn, so it reads 60 however few there are.
func (a *App) FPS() int {
	return a.fps
}
