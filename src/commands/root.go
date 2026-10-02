package commands

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mt-shihab26/orivo/src/app"
	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/systems/ipc"
	"github.com/mt-shihab26/orivo/src/systems/logx"
	"github.com/mt-shihab26/orivo/src/systems/paths"
	"github.com/mt-shihab26/orivo/src/systems/sessions"
	"github.com/mt-shihab26/orivo/src/systems/store"
	"github.com/mt-shihab26/orivo/src/systems/timer"
	"github.com/mt-shihab26/orivo/src/systems/todos"
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

	log, err := sessions.Open(paths.Sessions())
	if err != nil {
		logx.Error("failed to read %s: %v", paths.Sessions(), err)
	}

	world := &core.World{
		Config:   cfg,
		Timer:    timer.New(cfg.Timer, store.Load(paths.Store()), log),
		Todos:    todos.Cache{Path: paths.TodoistCache()},
		Sessions: log,
	}

	ipc.Serve(paths.Socket(), world.Timer)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		world.Quit()
	}()

	a := app.New(world)
	defer a.Close()
	a.Run()
	return nil
}
