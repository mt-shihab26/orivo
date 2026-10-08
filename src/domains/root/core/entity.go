package core

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
