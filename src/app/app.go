package app

import (
	"os"
	"os/signal"
	"slices"
	"sync/atomic"
	"syscall"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/entities"
)

type App struct {
	fonts    *core.Fonts
	input    *core.Input
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
		input: &core.Input{},
	}

	pomodoro := entities.NewPomodoro(cfg.Timer, a.input)

	a.entities = []core.Entity{
		pomodoro,
		entities.NewTopBar(a.input, a.fonts, cfg.ShowFPS, a.Quit),
		entities.NewSessionBar(a.fonts, pomodoro),
		entities.NewClock(a.fonts, pomodoro),
		entities.NewTodoLabel(a.input, a.fonts, pomodoro),
		entities.NewHints(a.fonts),
		entities.NewTodoPicker(a.input, a.fonts, pomodoro),
		entities.NewReduceDialog(a.input, a.fonts, pomodoro),
	}

	a.quitOnSignal()
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
	keys := core.ReadKeys()
	modal := a.openModal()

	for _, entity := range a.entities {
		a.input.Keys = nil
		if modal == nil || entity == modal {
			a.input.Keys = keys
		}
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

func (a *App) openModal() core.Entity {
	for _, entity := range a.entities {
		if modal, ok := entity.(core.Modal); ok && modal.IsOpen() {
			return entity
		}
	}
	return nil
}

func (a *App) quitOnSignal() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		a.Quit()
	}()
}
