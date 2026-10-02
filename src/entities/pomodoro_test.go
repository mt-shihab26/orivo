package entities

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/core"
)

type fixture struct {
	*Pomodoro
	dir      string
	clock    time.Time
	notified []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{dir: t.TempDir(), clock: time.Date(2026, 10, 2, 9, 0, 0, 0, time.Local)}
	f.open()
	return f
}

func (f *fixture) open() {
	cfg := config.Default()
	cfg.Timer.LongBreakInterval = 2

	world := &core.World{Config: cfg}
	f.Pomodoro = newPomodoro(world, filepath.Join(f.dir, "store.json"), filepath.Join(f.dir, "sessions.jsonl"))
	f.Pomodoro.now = func() time.Time { return f.clock }
	f.Pomodoro.OnPhaseEnd = func(summary, _ string) { f.notified = append(f.notified, summary) }
}

func (f *fixture) press(ch rune) {
	f.world.Keys = []core.Key{{Ch: ch}}
	f.Update(0)
	f.world.Keys = nil
}

func (f *fixture) wait(d time.Duration) {
	f.clock = f.clock.Add(d)
	f.Update(float32(d.Seconds()))
}

func TestCountsDownOnlyWhileRunning(t *testing.T) {
	f := newFixture(t)

	f.wait(time.Minute)
	if got := f.Remaining(); got != 25*time.Minute {
		t.Fatalf("paused timer moved: %v", got)
	}

	f.press(' ')
	f.wait(10 * time.Minute)
	f.press(' ')
	f.wait(time.Hour)

	if f.Running || f.Remaining() != 15*time.Minute {
		t.Fatalf("got running=%v remaining=%v, want paused at 15m", f.Running, f.Remaining())
	}
}

func TestWorkRollsIntoBreakThenWaitsForWork(t *testing.T) {
	f := newFixture(t)

	f.press(' ')
	f.wait(25 * time.Minute)

	if f.Phase != Break || !f.Running || f.Remaining() != 5*time.Minute {
		t.Fatalf("after work: phase=%v running=%v remaining=%v", f.Phase, f.Running, f.Remaining())
	}
	if f.SessionsToday() != 1 {
		t.Fatalf("sessions today = %d, want 1", f.SessionsToday())
	}
	if f.world.Color != Break.Color() {
		t.Fatalf("world color did not follow the phase")
	}

	f.wait(5 * time.Minute)

	if f.Phase != Work || f.Running || f.Remaining() != 25*time.Minute {
		t.Fatalf("after break: phase=%v running=%v remaining=%v", f.Phase, f.Running, f.Remaining())
	}
	if f.SessionsToday() != 1 {
		t.Fatalf("a break counted as a session: %d", f.SessionsToday())
	}

	want := []string{"Work Session Complete", "Break Complete"}
	if !slices.Equal(f.notified, want) {
		t.Fatalf("notified %v, want %v", f.notified, want)
	}
}

func TestLongBreakFollowsEveryIntervalOfWork(t *testing.T) {
	f := newFixture(t)

	f.press('n')
	if f.Phase != Break {
		t.Fatalf("after 1 session: %v", f.Phase)
	}
	f.press('n')
	f.press('n')
	if f.Phase != LongBreak {
		t.Fatalf("after 2 sessions: %v", f.Phase)
	}
}

func TestResetRestoresTheFullPhase(t *testing.T) {
	f := newFixture(t)

	f.press(' ')
	f.wait(10 * time.Minute)
	f.press('r')

	if f.Running || f.Remaining() != 25*time.Minute {
		t.Fatalf("got running=%v remaining=%v, want paused at 25m", f.Running, f.Remaining())
	}
}

func TestReduceTakesTimeOffAndStopsAtZero(t *testing.T) {
	f := newFixture(t)

	f.Reduce(20 * time.Minute)
	if got := f.Remaining(); got != 5*time.Minute {
		t.Fatalf("remaining = %v, want 5m", got)
	}

	f.Reduce(time.Hour)
	if got := f.Remaining(); got != 0 {
		t.Fatalf("remaining = %v, want 0", got)
	}
}

func TestEachTodoKeepsItsOwnRemainingTime(t *testing.T) {
	f := newFixture(t)

	f.SetTodo("a", "Write report")
	f.press(' ')
	f.wait(10 * time.Minute)

	f.SetTodo("b", "Review PR")
	if f.Remaining() != 25*time.Minute || !f.Running {
		t.Fatalf("fresh todo: running=%v remaining=%v", f.Running, f.Remaining())
	}
	f.wait(time.Minute)

	f.SetTodo("a", "Write report")
	if got := f.Remaining(); got != 15*time.Minute {
		t.Fatalf("todo a remaining = %v, want 15m", got)
	}
}

func TestSessionsAreRecordedAgainstTheTodo(t *testing.T) {
	f := newFixture(t)

	f.SetTodo("a", "Write report")
	f.press(' ')
	f.wait(25 * time.Minute)

	if stat := f.Stat("a"); stat.Sessions != 1 || stat.Secs != 25*60 {
		t.Fatalf("stat = %+v, want 1 session of 25 min", stat)
	}
}

func TestStateSurvivesRestart(t *testing.T) {
	f := newFixture(t)

	f.SetTodo("a", "Write report")
	f.press(' ')
	f.wait(25 * time.Minute)
	f.wait(2 * time.Minute)
	f.Close()

	f.open()

	if f.Phase != Break || f.Running || f.Remaining() != 3*time.Minute {
		t.Fatalf("restored: phase=%v running=%v remaining=%v", f.Phase, f.Running, f.Remaining())
	}
	if f.TodoID != "a" || f.TodoText != "Write report" {
		t.Fatalf("restored todo: %q %q", f.TodoID, f.TodoText)
	}
	if f.SessionsToday() != 1 || f.Stat("a").Sessions != 1 {
		t.Fatalf("restored history: today=%d stat=%+v", f.SessionsToday(), f.Stat("a"))
	}
}
