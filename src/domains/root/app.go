package root

import (
	"os"
	"os/signal"
	"slices"
	"sync/atomic"
	"syscall"
	"time"

	gui "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/domains/root/core"
	"orivo/src/domains/root/widgets/session_bar"
	"orivo/src/domains/root/widgets/top_bar"
	"orivo/src/systems/config"
	"orivo/src/systems/logx"
	"orivo/src/systems/theme"
)

const heartbeat = time.Second

type App struct {
	fonts   *core.Fonts
	widgets []core.Widget
	quit    atomic.Bool
	themes  *theme.Watcher
	dirty   bool

	// Frames drawn in the last full second, and so far in this one.
	fps, draws int
	second     time.Time
}

func New(cfg config.Config) *App {
	rl.SetTraceLogLevel(rl.LogWarning)
	// Hyprland draws no title bars, so libdecor would only cost time and memory.
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" {
		disableLibdecor()
	}
	logx.Echo()
	why := pinPlatform()
	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagMsaa4xHint)
	rl.InitWindow(core.BaseWidth, core.BaseHeight, "Orivo")
	logx.Info("window: %s natively on %s, as %s", platformName(), osName(), why)
	watchClicks()
	rl.SetWindowMinSize(420, 360)
	rl.SetExitKey(0)
	// PollInputEvents and EndDrawing wait for input; wake ends the wait early.
	rl.EnableEventWaiting()
	setWindowOpen(true)
	core.Wake = wake
	// raygui fills its style the first time it is read; BeginGUI then
	// overrides it from the theme every frame.
	gui.LoadStyleDefault()

	a := &App{
		fonts: core.NewFonts(cfg.Font),
	}
	a.loadTheme()
	core.Redraw = a.redraw

	a.widgets = []core.Widget{
		top_bar.New(a.fonts, cfg.ShowFPS, a.FPS, a.Quit),
		session_bar.New(cfg.Timer, a.fonts),
	}
	for _, widget := range a.widgets {
		widget.Attach()
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
	for _, widget := range slices.Backward(a.widgets) {
		widget.Detach()
	}
	a.fonts.Close()
	setWindowOpen(false)
	rl.CloseWindow()
}

// redraw runs on the render loop's goroutine, from a click while drawing.
func (a *App) redraw() {
	a.dirty = true
	wake()
}

func (a *App) Quit() {
	a.quit.Store(true)
	wake()
}

// Run sleeps until there is input, until a widget is due to change on its
// own, or for a second at most, and draws only when something on screen may
// have changed: on input, when a widget says it changed, and at least once
// a second for anything else. A paused timer then costs almost no CPU.
func (a *App) Run() {
	last := time.Now()
	var drawn time.Time
	for !rl.WindowShouldClose() && !a.quit.Load() {
		afterPoll()
		scaleMouse()
		logClicks()
		now := time.Now()
		a.Update(float32(now.Sub(last).Seconds()))
		last = now
		if now.Sub(a.second) >= time.Second {
			a.fps, a.draws, a.second = a.draws, 0, now
		}

		alarm := time.AfterFunc(a.next(), wake)
		if a.dirty || input() || a.changed() || now.Sub(drawn) >= heartbeat {
			// EndDrawing waits for input; there is no frame rate to wait out.
			// Cleared first, as a click while drawing asks for one more frame.
			a.dirty = false
			a.Draw()
			drawn = now
			a.draws++
		} else {
			rl.PollInputEvents()
		}
		alarm.Stop()
	}
}

// next returns how long the window may wait for input: until the soonest
// widget is due to change, and never longer than a second.
func (a *App) next() time.Duration {
	wait := heartbeat
	for _, widget := range a.widgets {
		if t, ok := widget.(core.Ticker); ok {
			if next := t.Next(); next > 0 {
				wait = min(wait, next)
			}
		}
	}
	return wait
}

func (a *App) changed() bool {
	for _, widget := range a.widgets {
		if c, ok := widget.(core.Changer); ok && c.Changed() {
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

	for _, widget := range a.widgets {
		widget.Update(dt)
	}
}

func (a *App) Draw() {
	a.fonts.Ensure(core.CurrentScreen().Scale)

	rl.BeginDrawing()
	defer rl.EndDrawing()
	rl.ClearBackground(core.ColorBackground)
	core.BeginGUI(a.fonts)

	for _, widget := range a.widgets {
		widget.Draw()
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
