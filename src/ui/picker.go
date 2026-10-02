package ui

import (
	"errors"
	"fmt"
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/sessions"
	"github.com/mt-shihab26/orivo/src/todos"
)

type pickerAction int

const (
	pickerNone pickerAction = iota
	pickerCancel
	// pickerSelect means the user confirmed; the picker's current todo is the choice.
	pickerSelect
)

// todoPicker is the overlay for choosing which overdue or due-today todo the
// timer works on.
type todoPicker struct {
	// Overdue todos followed by today's; the first `overdue` entries are overdue.
	todos   []todos.Todo
	overdue int
	stats   map[string]sessions.Stat
	// Why the lists could not be loaded, nil when they were.
	err error
	// Index of the highlighted todo.
	cursor int
	// The todo currently assigned to the timer, shown filled in the phase color.
	selectedID string
	// Where each visible todo was last drawn, for mouse clicks.
	hits []pickerHit
}

type pickerHit struct {
	rect  rl.Rectangle
	index int
}

func newTodoPicker(lists todos.Lists, err error, log *sessions.Log, selectedID string) *todoPicker {
	p := &todoPicker{
		todos:      append(lists.Overdue, lists.Today...),
		overdue:    len(lists.Overdue),
		stats:      map[string]sessions.Stat{},
		err:        err,
		selectedID: selectedID,
	}
	for i, todo := range p.todos {
		p.stats[todo.ID] = log.Stat(todo.ID)
		// Start on the todo the timer is already pointed at.
		if todo.ID == selectedID {
			p.cursor = i
		}
	}
	return p
}

func (p *todoPicker) handle(k key) pickerAction {
	switch {
	case k.ch == 'j' || k.code == rl.KeyDown:
		p.move(1)
	case k.ch == 'k' || k.code == rl.KeyUp:
		p.move(-1)
	case k.code == rl.KeyEnter || k.code == rl.KeyKpEnter:
		if len(p.todos) == 0 {
			return pickerCancel
		}
		return pickerSelect
	case k.code == rl.KeyEscape:
		return pickerCancel
	}
	return pickerNone
}

// handleMouse scrolls with the wheel and selects the todo that was clicked.
func (p *todoPicker) handleMouse() pickerAction {
	if wheel := rl.GetMouseWheelMove(); wheel > 0 {
		p.move(-1)
	} else if wheel < 0 {
		p.move(1)
	}

	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		pos := rl.GetMousePosition()
		for _, hit := range p.hits {
			if rl.CheckCollisionPointRec(pos, hit.rect) {
				p.cursor = hit.index
				return pickerSelect
			}
		}
	}
	return pickerNone
}

func (p *todoPicker) move(delta int) {
	if len(p.todos) > 0 {
		p.cursor = min(max(p.cursor+delta, 0), len(p.todos)-1)
	}
}

func (p *todoPicker) current() todos.Todo {
	return p.todos[p.cursor]
}

// pickerRow is a display row: a section header, or the todo at index.
type pickerRow struct {
	header string
	index  int
}

func (p *todoPicker) rows() []pickerRow {
	var rows []pickerRow
	for i := range p.todos {
		if i == 0 && p.overdue > 0 {
			rows = append(rows, pickerRow{header: "Overdue"})
		}
		if i == p.overdue {
			rows = append(rows, pickerRow{header: "Today"})
		}
		rows = append(rows, pickerRow{index: i})
	}
	return rows
}

func (p *todoPicker) draw(f *fonts, w, h, s float32, col color.RGBA) {
	rl.DrawRectangle(0, 0, int32(w), int32(h), rl.Fade(colorBackground, 0.8))

	panel := rl.Rectangle{Width: min(600*s, w-32*s), Height: h - 64*s}
	panel.X, panel.Y = (w-panel.Width)/2, (h-panel.Height)/2
	drawPanel(panel, s, col)

	cx := w / 2
	f.body.drawCentered("Select Todo", cx, panel.Y+16*s, col)
	f.small.drawCentered("[j/k] Move   [Enter] Select   [Esc] Cancel",
		cx, panel.Y+panel.Height-30*s, colorDim)

	list := rl.Rectangle{
		X:      panel.X + 12*s,
		Y:      panel.Y + 54*s,
		Width:  panel.Width - 24*s,
		Height: panel.Height - 54*s - 44*s,
	}

	p.hits = p.hits[:0]
	if len(p.todos) == 0 {
		title, detail := p.emptyMessage()
		f.body.drawCentered(title, cx, list.Y+list.Height/2-24*s, colorText)
		f.small.drawCentered(f.small.fit(detail, list.Width), cx, list.Y+list.Height/2+8*s, colorDim)
		return
	}

	rows := p.rows()
	rowHeight := 32 * s
	visible := max(int(list.Height/rowHeight), 1)

	// Keep the cursor's row centered while there is room to scroll.
	cursorRow := 0
	for i, row := range rows {
		if row.header == "" && row.index == p.cursor {
			cursorRow = i
		}
	}
	start := min(max(cursorRow-visible/2, 0), max(len(rows)-visible, 0))
	end := min(start+visible, len(rows))

	for i, row := range rows[start:end] {
		rect := rl.Rectangle{
			X:      list.X,
			Y:      list.Y + float32(i)*rowHeight,
			Width:  list.Width,
			Height: rowHeight - 2*s,
		}
		textY := rect.Y + (rect.Height-f.body.size)/2

		if row.header != "" {
			f.small.draw(row.header, rect.X+8*s, rect.Y+(rect.Height-f.small.size)/2, colorDim)
			continue
		}
		p.hits = append(p.hits, pickerHit{rect: rect, index: row.index})

		todo := p.todos[row.index]
		isCursor := row.index == p.cursor
		isSelected := todo.ID == p.selectedID

		textColor, noteColor := colorText, colorDim
		switch {
		case isSelected:
			rl.DrawRectangleRounded(rect, 0.3, 6, col)
			textColor, noteColor = colorBackground, colorBackground
		case isCursor:
			rl.DrawRectangleRounded(rect, 0.3, 6, colorTrack)
			textColor = col
		}

		prefix := "  "
		if isCursor {
			prefix = "> "
		}

		// Session totals, and the due date for overdue todos, sit on the right.
		note := ""
		if stat := p.stats[todo.ID]; stat.Sessions > 0 {
			note = fmt.Sprintf("%d× %dm", stat.Sessions, stat.Secs/60)
		}
		if row.index < p.overdue {
			if note != "" {
				note += "  ·  "
			}
			note += todo.Due.Format("Jan 2")
		}
		noteWidth := float32(0)
		if note != "" {
			noteWidth = f.small.width(note) + 16*s
			f.small.draw(note, rect.X+rect.Width-noteWidth+4*s,
				rect.Y+(rect.Height-f.small.size)/2, noteColor)
		}

		label := f.body.fit(prefix+todo.Text, rect.Width-16*s-noteWidth)
		f.body.draw(label, rect.X+8*s, textY, textColor)
	}
}

func (p *todoPicker) emptyMessage() (title, detail string) {
	switch {
	case errors.Is(p.err, todos.ErrNoCache):
		return "No Todoist todos cached yet",
			"Todos due today and overdue will show up here."
	case p.err != nil:
		return "Could not read the Todoist cache", p.err.Error()
	default:
		return "Nothing due", "No todos are overdue or due today."
	}
}
