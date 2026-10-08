package todoist

import "orivo/src/systems/todos"

// Sync caches the due todos and queues the removal of the tag from those not
// worked on today, so a tag left from an earlier day, or kept by a recurring
// task that was completed, does not show yesterday's count.
func Sync(authPath, cachePath, outboxPath string, workedToday func(id string) bool) ([]todos.Todo, error) {
	token, err := NewOAuth().Token(authPath)
	if err != nil {
		return nil, err
	}

	all, tagged, err := NewClient(token).DueTodos()
	if err != nil {
		return nil, err
	}

	if err := (todos.Cache{Path: cachePath}).Write(all); err != nil {
		return nil, err
	}

	var stale []string
	for _, id := range tagged {
		if !workedToday(id) {
			stale = append(stale, id)
		}
	}
	if err := (Outbox{Path: outboxPath}).Clear(stale); err != nil {
		return nil, err
	}
	return all, nil
}
