package paths

import (
	"os"
	"path/filepath"
)

const app = "orivo"

var dev bool

func UseDev() {
	dev = true
}

func Config() string {
	return filepath.Join(configBase(), "config.toml")
}

func Log() string {
	return filepath.Join(StateDir(), "orivo.log")
}

func Store() string {
	return filepath.Join(StateDir(), "store.json")
}

func Sessions() string {
	return filepath.Join(StateDir(), "sessions.jsonl")
}

func TodoistCache() string {
	return filepath.Join(StateDir(), "todoist.json")
}

func Socket() string {
	return filepath.Join(StateDir(), "orivo.sock")
}

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
