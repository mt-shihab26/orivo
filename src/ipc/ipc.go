// Package ipc serves the live timer status over a Unix domain socket, so
// external tools (e.g. an Omarchy bar widget) can query it without reading
// store.json — whether the timer is running is only ever known in memory.
//
// Protocol: a client connects, orivo writes one JSON line describing the
// current status, then closes the connection. No request body is read.
package ipc

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"

	"github.com/mt-shihab26/orivo/src/logx"
	"github.com/mt-shihab26/orivo/src/timer"
)

type status struct {
	Phase            string  `json:"phase"`
	Label            string  `json:"label"`
	IsRunning        bool    `json:"is_running"`
	RemainingMillis  int64   `json:"remaining_millis"`
	TodoID           *string `json:"todo_id"`
	TodoText         *string `json:"todo_text"`
	SessionsToday    int     `json:"sessions_today"`
	DailySessionGoal int     `json:"daily_session_goal"`
}

// Serve starts answering connections on the socket at path in the background.
func Serve(path string, state *timer.State) {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o700)
	// Restrict the whole state directory to the owning user. This also closes
	// the socket's own creation race: Listen creates the socket file at the
	// umask-derived mode before it can be chmodded, but a non-traversable
	// parent means no other local account can reach it during that window.
	_ = os.Chmod(dir, 0o700)

	// Never take the socket away from an instance that is still serving it:
	// binding over a live socket leaves the first instance with a listener
	// nobody can reach. Connecting is the only way to tell a live socket from
	// one a crashed process left behind — they look identical on disk.
	if conn, err := net.Dial("unix", path); err == nil {
		conn.Close()
		logx.Warn("ipc: another orivo instance already serves %s; skipping IPC", path)
		return
	}

	// Nothing is listening, so clear the stale entry — but only when it really
	// is a socket, rather than deleting an unrelated file that sits there.
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			logx.Error("ipc: %s exists and is not a socket; refusing to replace it", path)
			return
		}
		_ = os.Remove(path)
	}

	listener, err := net.Listen("unix", path)
	if err != nil {
		logx.Error("ipc: failed to bind %s: %v", path, err)
		return
	}
	if err := os.Chmod(path, 0o600); err != nil {
		logx.Warn("ipc: failed to restrict permissions on %s: %v", path, err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				logx.Warn("ipc: accept failed: %v", err)
				return
			}
			respond(conn, state)
		}
	}()
}

// respond writes one JSON status line describing the current timer state.
func respond(conn net.Conn, state *timer.State) {
	defer conn.Close()

	snap := state.Snapshot()
	st := status{
		Phase:            snap.Phase.Key(),
		Label:            snap.Phase.Label(),
		IsRunning:        snap.Running,
		RemainingMillis:  snap.Remaining.Milliseconds(),
		SessionsToday:    snap.SessionsToday,
		DailySessionGoal: snap.DailyGoal,
	}
	if snap.TodoID != "" {
		st.TodoID = &snap.TodoID
		st.TodoText = &snap.TodoText
	}

	if err := json.NewEncoder(conn).Encode(st); err != nil {
		logx.Warn("ipc: write failed: %v", err)
	}
}
