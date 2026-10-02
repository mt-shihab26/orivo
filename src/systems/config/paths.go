package config

import (
	"os"
	"path/filepath"
)

const app = "orivo"

var dev bool

func UseDev() {
	dev = true
}

func ConfigPath() string {
	return filepath.Join(dir(".config"), "config.toml")
}

func Log() string          { return state("orivo.log") }
func Store() string        { return state("store.json") }
func Sessions() string     { return state("sessions.jsonl") }
func TodoistCache() string { return state("todoist.txt") }
func TodoistAuth() string  { return state("todoist-auth.json") }
func Socket() string       { return state("orivo.sock") }

func state(name string) string {
	return filepath.Join(dir(".local/state"), name)
}

func dir(base string) string {
	if dev {
		return ".dev"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, base, app)
}
