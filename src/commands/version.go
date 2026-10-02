package commands

import "fmt"

type Version struct {
	version string
}

func (c *Version) Name() string {
	return "version"
}

func (c *Version) Summary() string {
	return "Print the version"
}

func (c *Version) Run(args []string) error {
	fmt.Println("orivo", c.version)
	return nil
}
