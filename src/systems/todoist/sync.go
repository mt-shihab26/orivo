package todoist

import (
	"slices"

	"orivo/src/systems/files"
	"orivo/src/systems/todos"
)

// Sync caches the due todos and queues the removal of the tag from those not
// worked on today, so a tag left from an earlier day, or kept by a recurring
// task that was completed, does not show yesterday's count. A todo done in
// orivo but not yet completed in Todoist stays out of the cache.
func Sync(authPath, cachePath, outboxPath, donePath string, workedToday func(id string) bool) ([]todos.Todo, error) {
	token, err := NewOAuth().Token(authPath)
	if err != nil {
		return nil, err
	}

	all, tagged, err := NewClient(token).DueTodos()
	if err != nil {
		return nil, err
	}

	all, err = cacheDue(cachePath, donePath, all)
	if err != nil {
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

// cacheDue writes the todos to the cache, less those done in orivo and not
// yet completed in Todoist. The cache stays locked from reading the done ones
// to the write, so a todo marked done meanwhile is either left out here or
// taken out by its own removal after this write, never put back.
func cacheDue(cachePath, donePath string, all []todos.Todo) ([]todos.Todo, error) {
	unlock, err := files.Lock(cachePath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	done, err := Completions{Path: donePath}.Pending()
	if err != nil {
		return nil, err
	}
	all = slices.DeleteFunc(all, func(todo todos.Todo) bool { return slices.Contains(done, todo.ID) })
	return all, todos.Cache{Path: cachePath}.Write(all)
}
