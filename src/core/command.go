package core

type Command interface {
	Name() string
	Summary() string
	Run(args []string) error
}
