package timer

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/systems/phase"
	"github.com/mt-shihab26/orivo/src/systems/sessions"
	"github.com/mt-shihab26/orivo/src/systems/store"
)

type fixture struct {
	*State
	dir      string
	clock    time.Time
	notified []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{dir: t.TempDir(), clock: time.Date(2026, 10, 2, 9, 0, 0, 0, time.Local)}
	f.open(t)
	return f
}

func (f *fixture) open(t *testing.T) {
	t.Helper()
	log, err := sessions.Open(filepath.Join(f.dir, "sessions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default().Timer
	cfg.LongBreakInterval = 2

	f.State = New(cfg, store.Load(filepath.Join(f.dir, "store.json")), log)
	f.State.now = func() time.Time { return f.clock }
	f.State.OnPhaseEnd(func(summary, _ string) { f.notified = append(f.notified, summary) })
}

func (f *fixture) wait(d time.Duration) {
	f.clock = f.clock.Add(d)
	f.Tick()
}

func TestCountsDownOnlyWhileRunning(t *testing.T) {
	f := newFixture(t)

	f.wait(time.Minute)
	if got := f.Snapshot().Remaining; got != 25*time.Minute {
		t.Fatalf("paused timer moved: %v", got)
	}

	f.Toggle()
	f.wait(10 * time.Minute)
	f.Toggle()
	f.wait(time.Hour)

	snap := f.Snapshot()
	if snap.Running || snap.Remaining != 15*time.Minute {
		t.Fatalf("got running=%v remaining=%v, want paused at 15m", snap.Running, snap.Remaining)
	}
}

func TestWorkRollsIntoBreakThenWaitsForWork(t *testing.T) {
	f := newFixture(t)

	f.Toggle()
	f.wait(25 * time.Minute)

	snap := f.Snapshot()
	if snap.Phase != phase.Break || !snap.Running || snap.Remaining != 5*time.Minute {
		t.Fatalf("after work: %+v", snap)
	}
	if snap.SessionsToday != 1 {
		t.Fatalf("sessions today = %d, want 1", snap.SessionsToday)
	}

	f.wait(5 * time.Minute)

	snap = f.Snapshot()
	if snap.Phase != phase.Work || snap.Running || snap.Remaining != 25*time.Minute {
		t.Fatalf("after break: %+v", snap)
	}
	if snap.SessionsToday != 1 {
		t.Fatalf("a break counted as a session: %d", snap.SessionsToday)
	}

	want := []string{"Work Session Complete", "Break Complete"}
	if len(f.notified) != 2 || f.notified[0] != want[0] || f.notified[1] != want[1] {
		t.Fatalf("notified %v, want %v", f.notified, want)
	}
}

func TestLongBreakFollowsEveryIntervalOfWork(t *testing.T) {
	f := newFixture(t)

	f.Skip()
	if got := f.Snapshot().Phase; got != phase.Break {
		t.Fatalf("after 1 session: %v", got)
	}
	f.Skip()
	f.Skip()
	if got := f.Snapshot().Phase; got != phase.LongBreak {
		t.Fatalf("after 2 sessions: %v", got)
	}
}

func TestReduceTakesTimeOffAndStopsAtZero(t *testing.T) {
	f := newFixture(t)

	f.Reduce(20 * time.Minute)
	if got := f.Snapshot().Remaining; got != 5*time.Minute {
		t.Fatalf("remaining = %v, want 5m", got)
	}

	f.Reduce(time.Hour)
	if got := f.Snapshot().Remaining; got != 0 {
		t.Fatalf("remaining = %v, want 0", got)
	}
}

func TestEachTodoKeepsItsOwnRemainingTime(t *testing.T) {
	f := newFixture(t)

	f.SetTodo("a", "Write report")
	f.Toggle()
	f.wait(10 * time.Minute)

	f.SetTodo("b", "Review PR")
	if snap := f.Snapshot(); snap.Remaining != 25*time.Minute || !snap.Running {
		t.Fatalf("fresh todo: %+v", snap)
	}
	f.wait(time.Minute)

	f.SetTodo("a", "Write report")
	if got := f.Snapshot().Remaining; got != 15*time.Minute {
		t.Fatalf("todo a remaining = %v, want 15m", got)
	}
}

func TestSessionsAreRecordedAgainstTheTodo(t *testing.T) {
	f := newFixture(t)

	f.SetTodo("a", "Write report")
	f.Toggle()
	f.wait(25 * time.Minute)

	snap := f.Snapshot()
	if snap.Stat.Sessions != 1 || snap.Stat.Secs != 25*60 {
		t.Fatalf("stat = %+v, want 1 session of 25 min", snap.Stat)
	}
}

func TestStateSurvivesRestart(t *testing.T) {
	f := newFixture(t)

	f.SetTodo("a", "Write report")
	f.Toggle()
	f.wait(25 * time.Minute)
	f.wait(2 * time.Minute)
	f.Save()

	f.open(t)

	snap := f.Snapshot()
	if snap.Phase != phase.Break || snap.Running || snap.Remaining != 3*time.Minute {
		t.Fatalf("restored: %+v", snap)
	}
	if snap.TodoID != "a" || snap.TodoText != "Write report" || snap.SessionsToday != 1 {
		t.Fatalf("restored: %+v", snap)
	}
}
