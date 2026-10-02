package root

import (
	"os"
	"os/signal"
	"slices"
	"sync/atomic"
	"syscall"

	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/domains/root/core"
	"orivo/src/domains/root/entities/hints"
	"orivo/src/domains/root/entities/session_bar"
	"orivo/src/domains/root/entities/top_bar"
	"orivo/src/systems/config"
	"orivo/src/systems/logx"
	"orivo/src/systems/theme"
)

type App struct {
	fonts    *core.Fonts
	entities []core.Entity
	quit     atomic.Bool
	themes   *theme.Watcher
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
		top_bar.New(a.fonts, cfg.ShowFPS, a.Quit),
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

func (a *App) Run() {
	for !rl.WindowShouldClose() && !a.quit.Load() {
		a.Update(rl.GetFrameTime())
		a.Draw()
	}
}

func (a *App) Update(dt float32) {
	if a.themes != nil {
		select {
		case t := <-a.themes.Themes:
			core.ApplyTheme(t)
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
