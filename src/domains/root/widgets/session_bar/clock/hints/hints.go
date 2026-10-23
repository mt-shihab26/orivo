package hints

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/domains/root/core"
)

// Controls is what the hint buttons do, the same as their keys.
type Controls interface {
	Toggle()
	Reset()
	Skip()
	Done()
	Undo()
	PickTodo()
	ClearTodo()
	OpenReduce()
	Sync()
	ToggleMillis()
}

type hint struct {
	key, name string
	do        func(Controls)
}

// Two rows, the timer's own keys above those for the todo.
var rows = [][]hint{
	{
		{"Space", "Toggle", Controls.Toggle},
		{"r", "Reset", Controls.Reset},
		{"n", "Skip", Controls.Skip},
		{"c", "Done", Controls.Done},
		{"u", "Undo", Controls.Undo},
	},
	{
		{"t", "Todo", Controls.PickTodo},
		{"T", "Clear", Controls.ClearTodo},
		{"d", "Reduce", Controls.OpenReduce},
		{"s", "Sync", Controls.Sync},
		{"m", "Millis", Controls.ToggleMillis},
	},
}

// Height is how far above the bottom of the window the hints reach.
const Height = 70

type Hints struct {
	fonts    *core.Fonts
	controls Controls
}

func New(fonts *core.Fonts, controls Controls) *Hints {
	return &Hints{fonts: fonts, controls: controls}
}

func (h *Hints) Attach() {}

func (h *Hints) Detach() {}

func (h *Hints) Update(dt float32) {}

// Draw lays each row out as buttons centered at the foot of the window. A
// row too wide for the window drops the keys from its labels.
func (h *Hints) Draw() {
	screen := core.CurrentScreen()
	s := screen.Scale
	gap, height := 6*s, 22*s

	for i, row := range rows {
		labels := make([]string, len(row))
		width := rowWidth(h.fonts, row, labels, true)
		if width > screen.Width-24*s {
			width = rowWidth(h.fonts, row, labels, false)
		}

		x := (screen.Width - width) / 2
		y := screen.Height - (Height-6)*s + float32(i)*(height+4*s)
		for j, label := range labels {
			w := core.ButtonWidth(h.fonts, label)
			if core.Button(rl.Rectangle{X: x, Y: y, Width: w, Height: height}, label) {
				row[j].do(h.controls)
			}
			x += w + gap
		}
	}
}

// rowWidth fills labels for row, with or without the keys, and returns how
// wide the row of buttons is.
func rowWidth(fonts *core.Fonts, row []hint, labels []string, keys bool) float32 {
	s := core.CurrentScreen().Scale
	width := float32(0)
	for i, hint := range row {
		labels[i] = hint.name
		if keys {
			labels[i] = "[" + hint.key + "] " + hint.name
		}
		width += core.ButtonWidth(fonts, labels[i]) + 6*s
	}
	return width - 6*s
}
