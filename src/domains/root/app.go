package root

import (
	"slices"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/domains/root/core"
	"orivo/src/domains/root/entities/hints"
	"orivo/src/domains/root/entities/session_bar"
	"orivo/src/domains/root/entities/top_bar"
	"orivo/src/systems/config"
	"orivo/src/systems/signals"
)

type App struct {
	fonts    *core.Fonts
	entities []core.Entity
	quit     atomic.Bool
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

	a.entities = []core.Entity{
		top_bar.New(a.fonts, cfg.ShowFPS, a.Quit),
		hints.New(a.fonts),
		session_bar.New(cfg.Timer, a.fonts),
	}

	signals.OnInterrupt(a.Quit)
	return a
}

func (a *App) Close() {
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
