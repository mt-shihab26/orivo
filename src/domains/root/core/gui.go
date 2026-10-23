package core

import (
	"image/color"

	gui "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Every control drawn while a dialog is open is locked, so a click meant for
// the dialog never reaches what is behind it. A dialog unlocks the controls
// just before drawing its own, and is drawn last.
var modals int

func OpenModal()  { modals++ }
func CloseModal() { modals = max(modals-1, 0) }

// BeginGUI styles raygui from the theme and the Small font, and locks the
// controls while a dialog is open. App calls it at the start of each frame.
func BeginGUI(fonts *Fonts) {
	gui.SetFont(fonts.Small.font)
	gui.SetStyle(gui.DEFAULT, gui.TEXT_SIZE, gui.PropertyValue(fonts.Small.Size))
	gui.SetStyle(gui.DEFAULT, gui.TEXT_SPACING, gui.PropertyValue(fonts.Small.spacing))
	gui.SetStyle(gui.DEFAULT, gui.BORDER_WIDTH, 1)

	set := func(border, base, text color.RGBA, props ...gui.PropertyID) {
		for i, c := range []color.RGBA{border, base, text} {
			gui.SetStyle(gui.DEFAULT, props[i], gui.NewColorPropertyValue(c))
			gui.SetStyle(gui.BUTTON, props[i], gui.NewColorPropertyValue(c))
			gui.SetStyle(gui.LABEL, props[i], gui.NewColorPropertyValue(c))
		}
	}
	set(ColorTrack, ColorPanel, ColorDim, gui.BORDER_COLOR_NORMAL, gui.BASE_COLOR_NORMAL, gui.TEXT_COLOR_NORMAL)
	set(ColorDim, ColorTrack, ColorText, gui.BORDER_COLOR_FOCUSED, gui.BASE_COLOR_FOCUSED, gui.TEXT_COLOR_FOCUSED)
	set(ColorText, ColorTrack, ColorText, gui.BORDER_COLOR_PRESSED, gui.BASE_COLOR_PRESSED, gui.TEXT_COLOR_PRESSED)

	if modals > 0 {
		gui.Lock()
	} else {
		gui.Unlock()
	}
}

// Unlock lets the controls drawn next take clicks; an open dialog calls it
// before drawing its own.
func Unlock() {
	gui.Unlock()
}

// Button draws a button and reports whether it was clicked this frame.
func Button(rect rl.Rectangle, text string) bool {
	return clicked(gui.Button(rect, text))
}

// Hotspot reports whether rect was clicked this frame, drawing nothing, for
// things drawn by hand that also take clicks, such as the clock.
func Hotspot(rect rl.Rectangle) bool {
	return clicked(gui.LabelButton(rect, ""))
}

// ButtonWidth is how wide a button must be to fit text.
func ButtonWidth(fonts *Fonts, text string) float32 {
	return fonts.Small.Width(text) + 20*CurrentScreen().Scale
}

func clicked(yes bool) bool {
	if yes {
		Redraw()
	}
	return yes
}
