package commands

import (
	"slices"

	"orivo/src/domains/connect_todoist"
)

type ConnectTodoist struct{}

func (c *ConnectTodoist) Name() string {
	return "connect-todoist"
}

func (c *ConnectTodoist) Summary() string {
	return "Sign in to Todoist in your browser (--token to paste an API token instead)"
}

func (c *ConnectTodoist) Run(args []string) error {
	return connect_todoist.Run(slices.Contains(args, "--token"))
}
