package commands

import (
	"fmt"

	"orivo/src/domains/root"
	"orivo/src/systems/config"
)

type Root struct{}

func (c *Root) Name() string {
	return ""
}

func (c *Root) Summary() string {
	return "Open the timer window"
}

func (c *Root) Run(args []string) error {
	cfg, err := config.Load(config.ConfigPath())
	if err != nil {
		return fmt.Errorf("%s: %w", config.ConfigPath(), err)
	}
	a := root.New(cfg)
	defer a.Close()
	a.Run()
	return nil
}
