package app

import (
	"slices"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/entities/clock"
	"github.com/mt-shihab26/orivo/src/entities/hints"
	"github.com/mt-shihab26/orivo/src/entities/sessionbar"
	"github.com/mt-shihab26/orivo/src/entities/topbar"
	"github.com/mt-shihab26/orivo/src/systems/signals"
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

	timer := clock.New(cfg.Timer, a.fonts)

	a.entities = []core.Entity{
		topbar.New(a.fonts, cfg.ShowFPS, a.Quit),
		sessionbar.New(a.fonts, timer),
		hints.New(a.fonts),
		timer,
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
