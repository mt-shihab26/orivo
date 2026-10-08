package core

import "time"

type Entity interface {
	Close()
	Update(dt float32)
	Draw()
}

// Changer is an entity whose look can change with no input, such as a
// running clock. Changed reports whether it looks different from its last
// Draw, so the window is redrawn only then.
type Changer interface {
	Changed() bool
}

// Ticker is an entity that changes on its own at a known time, such as a
// running clock. Next returns how long until then, or 0 for never, so the
// window can wait for input until that moment.
type Ticker interface {
	Next() time.Duration
}

// Wake makes the window stop waiting for input and update now. It is safe
// to call from any goroutine, such as one bringing back a sync result.
var Wake = func() {}
