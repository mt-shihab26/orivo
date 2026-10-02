package core

type Entity interface {
	Close()
	Update(dt float32)
	Draw()
}

type Modal interface {
	Entity
	IsOpen() bool
}
