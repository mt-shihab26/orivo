package main

import (
	"fmt"
	"os"
	"slices"

	"orivo/src/commands"
	"orivo/src/systems/config/paths"
)

var version = "dev"

func main() {
	args := os.Args[1:]
	if i := slices.Index(args, "--dev"); i >= 0 {
		args = slices.Delete(args, i, i+1)
		paths.UseDev()
	}

	if err := commands.Route(version, args); err != nil {
		fmt.Fprintln(os.Stderr, "orivo:", err)
		os.Exit(1)
	}
}
