package commands

import (
	"errors"
	"fmt"

	"orivo/src/domains/root"
	"orivo/src/domains/root/entities/session_bar/clock/ipc"
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
	// Two windows would overwrite each other's timer and count sessions twice.
	if ipc.Running(config.Socket()) {
		return errors.New("orivo is already running")
	}

	cfg, err := config.Load(config.ConfigPath())
	if err != nil {
		return fmt.Errorf("%s: %w", config.ConfigPath(), err)
	}
	a := root.New(cfg)
	defer a.Close()
	a.Run()
	return nil
}
