// Package ui draws the timer window with raylib and turns input into timer
// actions.
package ui

import (
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/notify"
	"github.com/mt-shihab26/orivo/src/sessions"
	"github.com/mt-shihab26/orivo/src/timer"
	"github.com/mt-shihab26/orivo/src/todos"
)

const (
	// Window size the layout is designed for; everything scales from it.
	baseWidth  = 720
	baseHeight = 540
)

// key is one press: a typed character, or a non-printing key code.
type key struct {
	ch   rune
	code int32
}

// Keys that act without producing a character, and repeat while held down.
var controlKeys = []int32{
	rl.KeyEnter, rl.KeyKpEnter, rl.KeyEscape, rl.KeyBackspace, rl.KeyUp, rl.KeyDown,
}

type App struct {
	cfg    config.Config
	state  *timer.State
	source todos.Source
	log    *sessions.Log

	fonts   *fonts
	showFPS bool
	// Open todo picker overlay, nil when closed.
	picker *todoPicker
	// Open reduce-time dialog, nil when closed.
	reduce *reducePicker
	quit   atomic.Bool
}

func New(cfg config.Config, state *timer.State, source todos.Source, log *sessions.Log) *App {
	return &App{cfg: cfg, state: state, source: source, log: log, showFPS: cfg.ShowFPS}
}

// Quit asks the window to close; safe to call from another goroutine.
func (a *App) Quit() {
	a.quit.Store(true)
}

// Run opens the window and blocks until it is closed.
func (a *App) Run() {
	rl.SetTraceLogLevel(rl.LogWarning)
	// No vsync: on Wayland a vsynced swap blocks for as long as the window is
	// not shown (e.g. on another workspace), which freezes the loop and gets
	// the app flagged as not responding. The frame limiter paces it instead.
	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagMsaa4xHint)
	rl.InitWindow(baseWidth, baseHeight, "Orivo")
	defer rl.CloseWindow()
	rl.SetWindowMinSize(420, 360)
	rl.SetTargetFPS(60)
	// Escape closes dialogs; it must not also close the window.
	rl.SetExitKey(0)

	a.fonts = newFonts(a.cfg.Font)
	defer a.fonts.close()
	a.fonts.need(a.state.Snapshot().TodoText)

	rl.InitAudioDevice()
	defer rl.CloseAudioDevice()
	beep := newBeep()
	defer rl.UnloadSound(beep)

	a.state.OnPhaseEnd(func(summary, body string) {
		notify.Send(summary, body)
		rl.PlaySound(beep)
	})
	defer a.state.OnPhaseEnd(nil)

	for !rl.WindowShouldClose() && !a.quit.Load() {
		a.state.Tick()
		a.handleInput()
		a.draw()
	}
}

func (a *App) handleInput() {
	ctrl := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)

	// Read presses from the queue rather than the per-frame key state, which
	// misses a key pressed and released within a single frame.
	for code := rl.GetKeyPressed(); code != 0; code = rl.GetKeyPressed() {
		switch {
		case ctrl && (code == rl.KeyQ || code == rl.KeyC):
			a.Quit()
		case ctrl && code == rl.KeyF:
			a.showFPS = !a.showFPS
		case !ctrl && slices.Contains(controlKeys, code):
			a.handleKey(key{code: code})
		}
	}
	for _, code := range controlKeys {
		if !ctrl && rl.IsKeyPressedRepeat(code) {
			a.handleKey(key{code: code})
		}
	}
	for ch := rl.GetCharPressed(); ch != 0; ch = rl.GetCharPressed() {
		if !ctrl {
			a.handleKey(key{ch: ch})
		}
	}

	if a.picker != nil && a.picker.handleMouse() == pickerSelect {
		a.pickerSelect()
	}
}

func (a *App) handleKey(k key) {
	if a.picker != nil {
		switch a.picker.handle(k) {
		case pickerSelect:
			a.pickerSelect()
		case pickerCancel:
			a.picker = nil
		}
		return
	}

	if a.reduce != nil {
		switch a.reduce.handle(k) {
		case reduceApply:
			a.state.Reduce(a.reduce.amount())
			a.reduce = nil
		case reduceCancel:
			a.reduce = nil
		}
		return
	}

	switch k.ch {
	case ' ':
		a.state.Toggle()
	case 'r':
		a.state.Reset()
	case 'n':
		a.state.Skip()
	case 't':
		a.openPicker()
	case 'T':
		a.state.SetTodo("", "")
	case 'm':
		a.state.ToggleMillis()
	case 'd':
		a.reduce = &reducePicker{}
	}
}

func (a *App) openPicker() {
	lists, err := a.source.Load(time.Now())
	a.picker = newTodoPicker(lists, err, a.log, a.state.Snapshot().TodoID)
	for _, todo := range a.picker.todos {
		a.fonts.need(todo.Text)
	}
	if err != nil {
		a.fonts.need(err.Error())
	}
}

func (a *App) pickerSelect() {
	todo := a.picker.current()
	a.state.SetTodo(todo.ID, todo.Text)
	a.picker = nil
}

func (a *App) draw() {
	w, h := float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())
	s := min(w/baseWidth, h/baseHeight)
	// Whole steps of scale, so a drag-resize does not reload fonts every frame.
	s = float32(int(s*20)) / 20
	s = min(max(s, 0.5), 4)

	snap := a.state.Snapshot()
	col := phaseColor(snap.Phase)
	f := a.fonts
	f.ensure(s)
	cx := w / 2

	rl.BeginDrawing()
	defer rl.EndDrawing()
	rl.ClearBackground(colorBackground)

	hints := "^q quit   ^f fps"
	f.small.draw(hints, 14*s, 12*s, colorDim)
	if a.showFPS {
		fps := fmt.Sprintf("%d fps", rl.GetFPS())
		f.small.draw(fps, w-14*s-f.small.width(fps), 12*s, colorDim)
	}

	a.drawSessions(snap, cx, 42*s, w, s)

	// The ring takes whatever height is left between the header and footer.
	top, bottom := 92*s, h-96*s
	radius := max(min((bottom-top)/2-6*s, w*0.38), 40*s)
	a.drawClock(snap, cx, (top+bottom)/2, radius, s)

	todo := "No todo selected  [t] pick"
	if snap.TodoID != "" {
		todo = snap.TodoText
		if snap.Stat.Sessions > 0 {
			todo += fmt.Sprintf("  ·  %d sessions  ·  %d min", snap.Stat.Sessions, snap.Stat.Secs/60)
		}
	}
	f.body.drawCentered(f.body.fit(todo, w-40*s), cx, h-80*s, col)

	hint := "[Space] Toggle   [r] Reset   [n] Skip   [t] Todo   [T] Clear   [m] Millis   [d] Reduce"
	f.small.drawCentered(f.small.fit(hint, w-24*s), cx, h-34*s, colorDim)

	if a.picker != nil {
		a.picker.draw(f, w, h, s, col)
	}
	if a.reduce != nil {
		a.reduce.draw(f, w, h, s, col)
	}
}

// drawSessions draws "Session X / Y" over a progress bar toward the daily goal.
func (a *App) drawSessions(snap timer.Snapshot, cx, y, w, s float32) {
	col := phaseColor(snap.Phase)
	label := fmt.Sprintf("Session %d / %d", snap.SessionsToday, snap.DailyGoal)
	a.fonts.body.drawCentered(label, cx, y, col)

	bar := rl.Rectangle{X: w * 0.25, Y: y + 32*s, Width: w * 0.5, Height: 6 * s}
	ratio := min(float32(snap.SessionsToday)/float32(snap.DailyGoal), 1)

	rl.DrawRectangleRounded(bar, 1, 6, colorTrack)
	if ratio > 0 {
		bar.Width = max(bar.Width*ratio, bar.Height)
		rl.DrawRectangleRounded(bar, 1, 6, col)
	}
}

// drawClock draws the remaining time inside a ring that fills as the phase
// elapses, with the phase above the time and the status below it.
func (a *App) drawClock(snap timer.Snapshot, cx, cy, radius, s float32) {
	col := phaseColor(snap.Phase)
	f := a.fonts
	center := rl.Vector2{X: cx, Y: cy}

	thickness := max(7*s, 3)
	rl.DrawRing(center, radius-thickness, radius, 0, 360, 96, colorTrack)
	elapsed := 1 - float32(snap.Remaining)/float32(snap.Total)
	if elapsed > 0 {
		// Angles run clockwise from 3 o'clock; start the arc at 12.
		rl.DrawRing(center, radius-thickness, radius, -90, -90+360*min(elapsed, 1), 96, col)
	}

	text := clockText(snap.Remaining, snap.ShowMillis)
	// Size the digits to span most of the ring; monospace glyphs are about
	// 0.6 of the font size wide.
	inner := (radius - thickness) * 2 * 0.76
	size := min(inner/(0.6*float32(len(text))), radius*0.62)
	f.ensureClock(size)

	clock := f.clock
	if width := clock.width(text); width > inner {
		clock.size *= inner / width
	}
	clock.drawCentered(text, cx, cy-clock.size/2, col)

	f.body.drawCentered(snap.Phase.Label(), cx, cy-clock.size/2-f.body.size-10*s, col)

	status, statusColor := "Paused", colorDim
	if snap.Running {
		status, statusColor = "Running", col
	}
	f.body.drawCentered(status, cx, cy+clock.size/2+10*s, statusColor)
}

// clockText formats the remaining time as mm:ss, or mm:ss.cs with centiseconds.
func clockText(remaining time.Duration, showMillis bool) string {
	if showMillis {
		millis := remaining.Milliseconds()
		return fmt.Sprintf("%02d:%02d.%02d", millis/60000, millis/1000%60, millis%1000/10)
	}
	// Round up, so the clock reads the full duration at the start and only
	// shows 00:00 once the time is really up.
	secs := int64((remaining + time.Second - 1) / time.Second)
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}
