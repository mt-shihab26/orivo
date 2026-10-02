package todoist

import "orivo/src/systems/todos"

func Sync(authPath, cachePath string) ([]todos.Todo, error) {
	token, err := NewOAuth().Token(authPath)
	if err != nil {
		return nil, err
	}

	all, err := NewClient(token).DueTodos()
	if err != nil {
		return nil, err
	}

	if err := (todos.Cache{Path: cachePath}).Write(all); err != nil {
		return nil, err
	}
	return all, nil
}
