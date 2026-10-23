package ipc

import (
	"testing"
	"time"
)

func TestRunningStatusCountsDownSincePublished(t *testing.T) {
	published := time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC)

	running := Status{IsRunning: true, RemainingMillis: 60_000}
	if got := running.at(published, published.Add(700*time.Millisecond)).RemainingMillis; got != 59_300 {
		t.Errorf("running = %d, want 59300", got)
	}
	if got := running.at(published, published.Add(2*time.Minute)).RemainingMillis; got != 0 {
		t.Errorf("past the end = %d, want 0", got)
	}

	paused := Status{RemainingMillis: 60_000}
	if got := paused.at(published, published.Add(time.Minute)).RemainingMillis; got != 60_000 {
		t.Errorf("paused = %d, want 60000", got)
	}
}
