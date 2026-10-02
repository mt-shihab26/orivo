package commands

import "fmt"

type Help struct {
	commands []Command
}

func (c *Help) Name() string {
	return "help"
}

func (c *Help) Summary() string {
	return "Show this help"
}

func (c *Help) Run(args []string) error {
	fmt.Println("Usage: orivo [--dev] [command]")
	fmt.Println()
	fmt.Println("Commands:")
	for _, command := range append(c.commands, c) {
		name := command.Name()
		if name == "" {
			name = "(default)"
		}
		fmt.Printf("  %-16s  %s\n", name, command.Summary())
	}
	fmt.Println()
	fmt.Println("Options:")
	fmt.Printf("  %-16s  %s\n", "--dev", "Keep config and state under ./.dev instead of the system paths")
	return nil
}
