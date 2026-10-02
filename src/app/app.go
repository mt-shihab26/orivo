package app

import (
	"slices"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/entities"
)

type App struct {
	world    *core.World
	entities []core.Entity
}

func New(world *core.World) *App {
	rl.SetTraceLogLevel(rl.LogWarning)
	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagMsaa4xHint)
	rl.InitWindow(core.BaseWidth, core.BaseHeight, "Orivo")
	rl.SetWindowMinSize(420, 360)
	rl.SetTargetFPS(60)
	rl.SetExitKey(0)
	rl.InitAudioDevice()

	world.Fonts = core.NewFonts(world.Config.Font)

	return &App{
		world: world,
		entities: []core.Entity{
			entities.NewPomodoro(world),
			entities.NewTopBar(world),
			entities.NewSessionBar(world),
			entities.NewClock(world),
			entities.NewTodoLabel(world),
			entities.NewHints(world),
			entities.NewTodoPicker(world),
			entities.NewReduceDialog(world),
		},
	}
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

	var modal core.Entity
	for _, entity := range a.entities {
		if m, ok := entity.(core.Modal); ok && m.IsOpen() {
			modal = entity
		}
	}

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
