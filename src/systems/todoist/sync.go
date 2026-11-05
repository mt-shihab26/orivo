package todoist

import (
	"errors"
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

	completions := Completions{Path: donePath}
	done, err := completions.Pending()
	if err != nil {
		return nil, err
	}
	all = slices.DeleteFunc(all, func(todo todos.Todo) bool { return slices.Contains(done, todo.ID) })

	// A todo whose completion was undone is still closed in Todoist until
	// its reopening is sent, so it keeps the place the undo gave it.
	reopening, err := completions.Reopening()
	if err != nil {
		return nil, err
	}
	cache := todos.Cache{Path: cachePath}
	if len(reopening) > 0 {
		cached, err := cache.Read()
		if err != nil && !errors.Is(err, todos.ErrNoCache) {
			return nil, err
		}
		for _, todo := range cached {
			fetched := slices.ContainsFunc(all, func(t todos.Todo) bool { return t.ID == todo.ID })
			if slices.Contains(reopening, todo.ID) && !fetched {
				all = append(all, todo)
			}
		}
	}
	return all, cache.Write(all)
}
