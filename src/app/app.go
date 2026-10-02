package app

import (
	"os"
	"os/signal"
	"slices"
	"syscall"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/entities"
)

type App struct {
	world    *core.World
	entities []core.Entity
}

func New(cfg config.Config) *App {
	rl.SetTraceLogLevel(rl.LogWarning)
	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagMsaa4xHint)
	rl.InitWindow(core.BaseWidth, core.BaseHeight, "Orivo")
	rl.SetWindowMinSize(420, 360)
	rl.SetTargetFPS(60)
	rl.SetExitKey(0)
	rl.InitAudioDevice()

	world := &core.World{Config: cfg, Fonts: core.NewFonts(cfg.Font)}
	pomodoro := entities.NewPomodoro(world)

	a := &App{
		world: world,
		entities: []core.Entity{
			pomodoro,
			entities.NewTopBar(world),
			entities.NewSessionBar(world, pomodoro),
			entities.NewClock(world, pomodoro),
			entities.NewTodoLabel(world, pomodoro),
			entities.NewHints(world),
			entities.NewTodoPicker(world, pomodoro),
			entities.NewReduceDialog(world, pomodoro),
		},
	}
	a.quitOnSignal()
	return a
}

func (a *App) Close() {
	for _, entity := range slices.Backward(a.entities) {
		entity.Close()
	}
	a.world.Fonts.Close()
	rl.CloseAudioDevice()
	rl.CloseWindow()
}

func (a *App) Run() {
	for !rl.WindowShouldClose() && !a.world.ShouldQuit() {
		a.Update(rl.GetFrameTime())
		a.Draw()
	}
}

func (a *App) Update(dt float32) {
	a.world.Sync()
	keys := core.ReadKeys()
	modal := a.openModal()

	for _, entity := range a.entities {
		a.world.Keys = nil
		if modal == nil || entity == modal {
			a.world.Keys = keys
		}
		entity.Update(dt)
	}
}

func (a *App) Draw() {
	a.world.Sync()

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
		a.world.Quit()
	}()
}
