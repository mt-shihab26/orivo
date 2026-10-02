package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mt-shihab26/orivo/src/systems/paths"
	"github.com/mt-shihab26/orivo/src/systems/todoist"
)

type ConnectTodoist struct{}

func (c *ConnectTodoist) Name() string {
	return "connect-todoist"
}

func (c *ConnectTodoist) Summary() string {
	return "Save your Todoist API token"
}

func (c *ConnectTodoist) Run(args []string) error {
	fmt.Println("Copy your API token from Todoist: Settings > Integrations > Developer.")
	fmt.Print("Paste it here: ")

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	token := strings.TrimSpace(line)
	if token == "" {
		if err != nil {
			return errors.New("no token given")
		}
		return errors.New("the token is empty")
	}

	name, err := todoist.NewClient(token).User()
	if errors.Is(err, todoist.ErrTokenRejected) {
		return errors.New("Todoist rejected that token; nothing was saved")
	}
	if err != nil {
		return fmt.Errorf("could not reach Todoist: %w", err)
	}

	if err := todoist.SaveToken(paths.TodoistToken(), token); err != nil {
		return err
	}

	fmt.Printf("Connected as %s.\n", name)
	fmt.Println("Run `orivo sync-todoist` to fetch your todos.")
	return nil
}
