package commands

import (
	"errors"
	"fmt"
	"os"

	"orivo/src/domains/root"
	"orivo/src/domains/root/entities/session_bar/clock/ipc"
	"orivo/src/domains/root/entities/session_bar/clock/notify"
	"orivo/src/systems/config"
	"orivo/src/systems/logx"
)

type Root struct{}

func (c *Root) Name() string {
	return ""
}

func (c *Root) Summary() string {
	return "Open the timer window"
}

func (c *Root) Run(args []string) error {
	err := c.open()
	if err != nil {
		report(err)
	}
	return err
}

// report makes a startup failure visible when orivo was opened from the app
// menu, where nobody sees stderr.
func report(err error) {
	logx.Error("could not start: %v", err)
	if info, statErr := os.Stderr.Stat(); statErr == nil && info.Mode()&os.ModeCharDevice != 0 {
		return
	}
	if notifyErr := notify.Show("Could not start", err.Error()); notifyErr != nil {
		logx.Error("failed to send notification: %v", notifyErr)
	}
}

func (c *Root) open() error {
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
