package commands

import "orivo/src/domains/sync_todoist"

type SyncTodoist struct{}

func (c *SyncTodoist) Name() string {
	return "sync-todoist"
}

func (c *SyncTodoist) Summary() string {
	return "Fetch Work todos that are overdue or due today, and cache them"
}

func (c *SyncTodoist) Run(args []string) error {
	return sync_todoist.Run()
}
