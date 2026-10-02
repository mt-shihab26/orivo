package commands

import (
	"fmt"

	"orivo/src/domains/root"
	"orivo/src/systems/config"
	"orivo/src/systems/paths"
)

type Root struct{}

func (c *Root) Name() string {
	return ""
}

func (c *Root) Summary() string {
	return "Open the timer window"
}

func (c *Root) Run(args []string) error {
	cfg, err := config.Load(paths.Config())
	if err != nil {
		return fmt.Errorf("%s: %w", paths.Config(), err)
	}
	a := root.New(cfg)
	defer a.Close()
	a.Run()
	return nil
}
