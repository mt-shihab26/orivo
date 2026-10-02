package commands

import "fmt"

var aliases = map[string]string{
	"--version": "version",
	"-V":        "version",
	"--help":    "help",
	"-h":        "help",
}

func All(version string) []Command {
	all := []Command{
		&Root{},
		&ConnectTodoist{},
		&SyncTodoist{},
		&Version{version: version},
	}
	return append(all, &Help{commands: all})
}

func Route(version string, args []string) error {
	name := ""
	if len(args) > 0 {
		name, args = args[0], args[1:]
	}
	if alias, ok := aliases[name]; ok {
		name = alias
	}

	for _, command := range All(version) {
		if command.Name() == name {
			return command.Run(args)
		}
	}
	return fmt.Errorf("unknown command: %s\nrun `orivo help` for usage", name)
}
