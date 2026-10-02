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

func Serve(path string, state *timer.State) {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o700)
	_ = os.Chmod(dir, 0o700)

	if conn, err := net.Dial("unix", path); err == nil {
		conn.Close()
		logx.Warn("ipc: another orivo instance already serves %s; skipping IPC", path)
		return
	}

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
