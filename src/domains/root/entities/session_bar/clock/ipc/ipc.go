package ipc

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"

	"orivo/src/systems/logx"
)

type Status struct {
	Phase            string  `json:"phase"`
	Label            string  `json:"label"`
	IsRunning        bool    `json:"is_running"`
	RemainingMillis  int64   `json:"remaining_millis"`
	TodoID           *string `json:"todo_id"`
	TodoText         *string `json:"todo_text"`
	SessionsToday    int     `json:"sessions_today"`
	DailySessionGoal int     `json:"daily_session_goal"`
}

type Server struct {
	mu     sync.Mutex
	status Status
}

func Serve(path string) *Server {
	server := &Server{}

	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o700)
	_ = os.Chmod(dir, 0o700)

	if Running(path) {
		logx.Warn("ipc: another orivo instance already serves %s; skipping IPC", path)
		return server
	}

	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			logx.Error("ipc: %s exists and is not a socket; refusing to replace it", path)
			return server
		}
		_ = os.Remove(path)
	}

	listener, err := net.Listen("unix", path)
	if err != nil {
		logx.Error("ipc: failed to bind %s: %v", path, err)
		return server
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
			server.respond(conn)
		}
	}()
	return server
}

// Running reports whether another orivo window is serving the socket.
func Running(path string) bool {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (s *Server) Publish(status Status) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
}

func (s *Server) respond(conn net.Conn) {
	defer conn.Close()

	s.mu.Lock()
	status := s.status
	s.mu.Unlock()

	if err := json.NewEncoder(conn).Encode(status); err != nil {
		logx.Warn("ipc: write failed: %v", err)
	}
}
