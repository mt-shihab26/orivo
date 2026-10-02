package main

import (
	"fmt"
	"os"
	"os/signal"
	"slices"
	"syscall"

	"github.com/mt-shihab26/orivo/src/app"
	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/ipc"
	"github.com/mt-shihab26/orivo/src/logx"
	"github.com/mt-shihab26/orivo/src/paths"
	"github.com/mt-shihab26/orivo/src/sessions"
	"github.com/mt-shihab26/orivo/src/store"
	"github.com/mt-shihab26/orivo/src/timer"
	"github.com/mt-shihab26/orivo/src/todos"
)

var version = "dev"

const usage = `Usage: orivo [--dev] [command]

Commands:
  (default)   Open the timer window
  version     Print the version
  help        Show this help

Options:
  --dev       Keep config and state under ./.dev instead of the system paths
`

func main() {
	args := os.Args[1:]
	if i := slices.Index(args, "--dev"); i >= 0 {
		args = slices.Delete(args, i, i+1)
		paths.UseDev()
	}

	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "":
		if err := run(); err != nil {
			fmt.Fprintln(os.Stderr, "orivo:", err)
			os.Exit(1)
		}
	case "version", "--version", "-V":
		fmt.Println("orivo", version)
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\nrun `orivo help` for usage\n", cmd)
		os.Exit(2)
	}
}

func run() error {
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
		Todos:    todos.FileCache{Path: paths.TodoistCache()},
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
