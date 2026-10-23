package core

import "time"

// Widget is anything drawn in the window. A parent attaches its children
// when it is attached, updates and draws them every frame in between, and
// detaches them in reverse order when it is detached.
//
// The UI is immediate mode: Draw lays the widget out from its state each
// frame and asks raygui about clicks as it draws (see gui.go), so nothing
// keeps the positions of what was drawn. Update reads the keyboard and moves
// time along.
type Widget interface {
	// Attach takes what the widget holds while it is in the window, such as
	// a socket or a background goroutine.
	Attach()
	// Detach gives back what Attach took and saves anything worth keeping.
	Detach()
	Update(dt float32)
	Draw()
}

// Changer is a widget whose look can change with no input, such as a
// running clock. Changed reports whether it looks different from its last
// Draw, so the window is redrawn only then.
type Changer interface {
	Changed() bool
}

// Ticker is a widget that changes on its own at a known time, such as a
// running clock. Next returns how long until then, or 0 for never, so the
// window can wait for input until that moment.
type Ticker interface {
	Next() time.Duration
}

// Wake makes the window stop waiting for input and update now. It is safe
// to call from any goroutine, such as one bringing back a sync result.
var Wake = func() {}

// Redraw asks for one more frame, for a click handled while drawing: the
// frame it came in on was drawn from the state before it.
var Redraw = func() {}
