// Package paths resolves where orivo keeps its config and runtime state.
package paths

import (
	"os"
	"path/filepath"
)

const app = "orivo"

var dev bool

// UseDev keeps every file under ./.dev instead of the real system paths, so
// local state can be inspected or wiped freely while developing.
func UseDev() {
	dev = true
}

func Config() string {
	return filepath.Join(configBase(), "config.toml")
}

func Log() string {
	return filepath.Join(StateDir(), "orivo.log")
}

// Store is the persisted timer state file.
func Store() string {
	return filepath.Join(StateDir(), "store.json")
}

// Sessions is the append-only log of completed sessions.
func Sessions() string {
	return filepath.Join(StateDir(), "sessions.jsonl")
}

// TodoistCache is the cached Todoist task list the todo picker reads from.
func TodoistCache() string {
	return filepath.Join(StateDir(), "todoist.json")
}

// Socket is the Unix domain socket that exposes live timer state (including
// whether it is running, which is never persisted) to external tools such as
// bar widgets.
func Socket() string {
	return filepath.Join(StateDir(), "orivo.sock")
}

// StateDir is the base directory for runtime state files.
func StateDir() string {
	if dev {
		return ".dev"
	}
	return filepath.Join(home(), ".local", "state", app)
}

func configBase() string {
	if dev {
		return ".dev"
	}
	return filepath.Join(home(), ".config", app)
}

func home() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "."
}
