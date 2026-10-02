package todopicker

import (
	"errors"
	"fmt"
	"image/color"
	"sort"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/entities/dialog"
	"github.com/mt-shihab26/orivo/src/systems/paths"
	"github.com/mt-shihab26/orivo/src/systems/sessions"
	"github.com/mt-shihab26/orivo/src/systems/todos"
)

type Timer interface {
	Accent() color.RGBA
	TodoID() string
	Stat(todoID string) sessions.Stat
	SetTodo(id, text string)
}

type TodoPicker struct {
	dialog.Dialog
	timer Timer

	todos      []todos.Todo
	overdue    int
	err        error
	cursor     int
	selectedID string
	hits       []pickerHit
}

type pickerHit struct {
	rect  rl.Rectangle
	index int
}

type pickerRow struct {
	header string
	index  int
}

func New(fonts *core.Fonts, timer Timer) *TodoPicker {
	return &TodoPicker{
		Dialog: dialog.New(fonts, "Select Todo", "[j/k] Move   [Enter] Select   [Esc] Cancel"),
		timer:  timer,
	}
}

func Split(all []todos.Todo, now time.Time) (overdue, today []todos.Todo) {
	y, m, d := now.Local().Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, time.Local)

	for _, todo := range all {
		switch {
		case todo.Due.Before(start):
			overdue = append(overdue, todo)
		case todo.Due.Equal(start):
			today = append(today, todo)
		}
	}

	sort.SliceStable(overdue, func(i, j int) bool {
		return overdue[i].Due.Before(overdue[j].Due)
	})
	return overdue, today
}

func (p *TodoPicker) Update(dt float32) {
	shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

	if !p.IsOpen() {
		if shift || !rl.IsKeyPressed(rl.KeyT) {
			return
		}

		all, err := todos.Cache{Path: paths.TodoistCache()}.Read()
		overdue, today := Split(all, time.Now())

		p.todos = append(overdue, today...)
		p.overdue = len(overdue)
		p.err = err
		p.selectedID = p.timer.TodoID()
		p.cursor = 0
		p.hits = nil

		for i, todo := range p.todos {
			p.Fonts.Need(todo.Text)
			if todo.ID == p.selectedID {
				p.cursor = i
			}
		}
		if err != nil {
			p.Fonts.Need(err.Error())
		}
		p.Show()
		return
	}

	p.Dialog.Update(dt)
	if !p.IsOpen() {
		return
	}

	move := 0
	for _, key := range []int32{rl.KeyJ, rl.KeyDown} {
		if rl.IsKeyPressed(key) || rl.IsKeyPressedRepeat(key) {
			move++
		}
	}
	for _, key := range []int32{rl.KeyK, rl.KeyUp} {
		if rl.IsKeyPressed(key) || rl.IsKeyPressedRepeat(key) {
			move--
		}
	}
	if wheel := rl.GetMouseWheelMove(); wheel > 0 {
		move--
	} else if wheel < 0 {
		move++
	}
	if len(p.todos) > 0 {
		p.cursor = min(max(p.cursor+move, 0), len(p.todos)-1)
	}

	picked := rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		pos := rl.GetMousePosition()
		for _, hit := range p.hits {
			if rl.CheckCollisionPointRec(pos, hit.rect) {
				p.cursor = hit.index
				picked = true
			}
		}
	}

	if picked && len(p.todos) > 0 {
		todo := p.todos[p.cursor]
		p.timer.SetTodo(todo.ID, todo.Text)
	}
	if picked {
		p.Hide()
	}
}

func (p *TodoPicker) rows() []pickerRow {
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

func (p *TodoPicker) Draw() {
	if !p.IsOpen() {
		return
	}
	screen := core.CurrentScreen()
	s := screen.Scale
	fonts := p.Fonts

	p.Accent = p.timer.Accent()
	p.Panel = rl.Rectangle{Width: min(600*s, screen.Width-32*s), Height: screen.Height - 64*s}
	p.Panel.X, p.Panel.Y = (screen.Width-p.Panel.Width)/2, (screen.Height-p.Panel.Height)/2
	p.Dialog.Draw()

	cx := screen.Width / 2
	list := rl.Rectangle{
		X:      p.Panel.X + 12*s,
		Y:      p.Panel.Y + 54*s,
		Width:  p.Panel.Width - 24*s,
		Height: p.Panel.Height - 54*s - 44*s,
	}

	p.hits = p.hits[:0]
	if len(p.todos) == 0 {
		title, detail := p.emptyMessage()
		fonts.Body.DrawCentered(title, cx, list.Y+list.Height/2-24*s, core.ColorText)
		fonts.Small.DrawCentered(fonts.Small.Fit(detail, list.Width), cx, list.Y+list.Height/2+8*s, core.ColorDim)
		return
	}

	rows := p.rows()
	rowHeight := 32 * s
	visible := max(int(list.Height/rowHeight), 1)

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
		if row.header != "" {
			fonts.Small.Draw(row.header, rect.X+8*s, rect.Y+(rect.Height-fonts.Small.Size)/2, core.ColorDim)
			continue
		}
		p.hits = append(p.hits, pickerHit{rect: rect, index: row.index})
		p.drawTodo(rect, row.index)
	}
}

func (p *TodoPicker) drawTodo(rect rl.Rectangle, index int) {
	screen := core.CurrentScreen()
	s := screen.Scale
	fonts := p.Fonts
	accent := p.timer.Accent()

	todo := p.todos[index]
	isCursor := index == p.cursor

	textColor, noteColor := core.ColorText, core.ColorDim
	switch {
	case todo.ID == p.selectedID:
		rl.DrawRectangleRounded(rect, 0.3, 6, accent)
		textColor, noteColor = core.ColorBackground, core.ColorBackground
	case isCursor:
		rl.DrawRectangleRounded(rect, 0.3, 6, core.ColorTrack)
		textColor = accent
	}

	note := ""
	if stat := p.timer.Stat(todo.ID); stat.Sessions > 0 {
		note = fmt.Sprintf("%d× %dm", stat.Sessions, stat.Secs/60)
	}
	if index < p.overdue {
		if note != "" {
			note += "  ·  "
		}
		note += todo.Due.Format("Jan 2")
	}
	noteWidth := float32(0)
	if note != "" {
		noteWidth = fonts.Small.Width(note) + 16*s
		fonts.Small.Draw(note, rect.X+rect.Width-noteWidth+4*s,
			rect.Y+(rect.Height-fonts.Small.Size)/2, noteColor)
	}

	prefix := "  "
	if isCursor {
		prefix = "> "
	}
	label := fonts.Body.Fit(prefix+todo.Text, rect.Width-16*s-noteWidth)
	fonts.Body.Draw(label, rect.X+8*s, rect.Y+(rect.Height-fonts.Body.Size)/2, textColor)
}

func (p *TodoPicker) emptyMessage() (title, detail string) {
	switch {
	case errors.Is(p.err, todos.ErrNoCache):
		return "No todos synced yet",
			"Run `orivo sync-todoist` to fetch them from Todoist."
	case p.err != nil:
		return "Could not read the Todoist cache", p.err.Error()
	default:
		return "Nothing due", "No todos are overdue or due today."
	}
}
