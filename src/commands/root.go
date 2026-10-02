package commands

import (
	"fmt"

	"github.com/mt-shihab26/orivo/src/app"
	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/systems/paths"
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
	a := app.New(cfg)
	defer a.Close()
	a.Run()
	return nil
}
