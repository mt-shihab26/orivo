package clock

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"orivo/src/domains/root/entities/session_bar/clock/phase"
	"orivo/src/systems/config"
)

type fixture struct {
	*Clock
	dir      string
	time     time.Time
	notified []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{dir: t.TempDir(), time: time.Date(2026, 10, 2, 9, 0, 0, 0, time.Local)}
	f.open()
	return f
}

func (f *fixture) open() {
	cfg := config.Default()
	cfg.Timer.LongBreakInterval = 2

	f.Clock = newClock(cfg.Timer, nil, filepath.Join(f.dir, "store.json"), filepath.Join(f.dir, "sessions"))
	f.Clock.now = func() time.Time { return f.time }
	f.Clock.notify = func(summary, _ string) { f.notified = append(f.notified, summary) }
	logSync = func(string, ...any) {}
}

func (f *fixture) start() {
	f.running = true
	f.startedAt = f.time
	f.phaseStartedAt = f.time
}

func (f *fixture) wait(d time.Duration) {
	f.time = f.time.Add(d)
	f.Update(float32(d.Seconds()))
}

func TestCountsDownOnlyWhileRunning(t *testing.T) {
	f := newFixture(t)

	f.wait(time.Minute)
	if got := f.Remaining(); got != 25*time.Minute {
		t.Fatalf("paused timer moved: %v", got)
	}

	f.start()
	f.wait(10 * time.Minute)

	if !f.running || f.Remaining() != 15*time.Minute {
		t.Fatalf("got running=%v remaining=%v, want running at 15m", f.running, f.Remaining())
	}
}

func TestWorkRollsIntoBreakThenWaitsForWork(t *testing.T) {
	f := newFixture(t)

	f.start()
	f.wait(25 * time.Minute)

	if f.phase != phase.Break || !f.running || f.Remaining() != 5*time.Minute {
		t.Fatalf("after work: phase=%v running=%v remaining=%v", f.phase, f.running, f.Remaining())
	}
	if f.SessionsToday() != 1 {
		t.Fatalf("sessions today = %d, want 1", f.SessionsToday())
	}
	if f.Accent() != phase.Break.Color() {
		t.Fatalf("accent did not follow the phase")
	}

	f.wait(5 * time.Minute)

	if f.phase != phase.Work || f.running || f.Remaining() != 25*time.Minute {
		t.Fatalf("after break: phase=%v running=%v remaining=%v", f.phase, f.running, f.Remaining())
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

	f.start()
	f.wait(25 * time.Minute)
	if f.phase != phase.Break {
		t.Fatalf("after 1 session: %v", f.phase)
	}
	f.wait(5 * time.Minute)

	f.start()
	f.wait(25 * time.Minute)
	if f.phase != phase.LongBreak {
		t.Fatalf("after 2 sessions: %v", f.phase)
	}
}

func TestOpenDialogDoesNotStopTheCountdown(t *testing.T) {
	f := newFixture(t)

	f.start()
	f.reduce.Show()
	f.wait(25 * time.Minute)

	if f.phase != phase.Break {
		t.Fatalf("phase = %v, want the break to have started", f.phase)
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
	f.start()
	f.wait(10 * time.Minute)

	f.SetTodo("b", "Review PR")
	if f.Remaining() != 25*time.Minute || !f.running {
		t.Fatalf("fresh todo: running=%v remaining=%v", f.running, f.Remaining())
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
	f.start()
	f.wait(25 * time.Minute)

	if stat := f.Stat("a"); stat.Sessions != 1 || stat.Secs != 25*60 {
		t.Fatalf("stat = %+v, want 1 session of 25 min", stat)
	}
}

func TestFinishedWorkSendsTheTodosProgress(t *testing.T) {
	f := newFixture(t)
	var sent []string
	f.progress = func(todoID, line string) { sent = append(sent, todoID+" "+line) }

	f.SetTodo("a", "Write report")
	f.start()
	f.wait(25 * time.Minute) // work ends
	f.wait(5 * time.Minute)  // the break ends, and must not send

	if len(sent) != 1 || sent[0] != "a (1, 25 min)" {
		t.Fatalf("sent = %q", sent)
	}
}

func TestEachDayStartsTheTodosProgressFromZero(t *testing.T) {
	f := newFixture(t)
	var sent []string
	f.progress = func(_, line string) { sent = append(sent, line) }

	f.SetTodo("a", "Write report")
	f.start()
	f.wait(25 * time.Minute)
	f.wait(5 * time.Minute)
	f.start()
	f.wait(25 * time.Minute)

	f.time = f.time.Add(24 * time.Hour)
	f.open()
	if f.SessionsToday() != 0 || f.Stat("a").Sessions != 0 {
		t.Fatalf("next day: today=%d stat=%+v, want both zero", f.SessionsToday(), f.Stat("a"))
	}

	f.progress = func(_, line string) { sent = append(sent, line) }
	f.phase = phase.Work
	f.start()
	f.wait(25 * time.Minute)

	want := []string{
		"(1, 25 min)",
		"(2, 50 min)",
		"(1, 25 min)",
	}
	if !slices.Equal(sent, want) {
		t.Fatalf("sent = %q, want %q", sent, want)
	}
}

func TestFinishedWorkIsSentAtOnce(t *testing.T) {
	f := newFixture(t)
	f.progress = func(string, string) {}
	flushes := 0
	f.flush = func() bool { flushes++; return true }

	f.wait(time.Minute)
	if flushes != 0 {
		t.Fatalf("flushed with nothing queued")
	}

	f.SetTodo("a", "Write report")
	f.start()
	f.wait(25 * time.Minute)
	if flushes != 1 {
		t.Fatalf("flushes = %d, want 1 as soon as work ends", flushes)
	}
}

func TestSendsAreAtLeastAMinuteApart(t *testing.T) {
	f := newFixture(t)
	var sent []time.Time
	f.flush = func() bool { sent = append(sent, f.time); return true }

	f.flushWanted = true
	f.wait(time.Second)
	f.flushWanted = true
	f.wait(30 * time.Second)
	if len(sent) != 1 {
		t.Fatalf("sent %d times within a minute", len(sent))
	}

	f.wait(29 * time.Second)
	if len(sent) != 1 {
		t.Fatalf("sent again after %v", f.time.Sub(sent[0]))
	}
	f.wait(time.Second)
	if len(sent) != 2 || sent[1].Sub(sent[0]) != time.Minute {
		t.Fatalf("sent = %v, want the second exactly a minute after the first", sent)
	}
}

func TestBusyFlushIsRetried(t *testing.T) {
	f := newFixture(t)
	f.progress = func(string, string) {}
	busy := true
	flushes := 0
	f.flush = func() bool {
		if busy {
			return false
		}
		flushes++
		return true
	}

	f.SetTodo("a", "Write report")
	f.start()
	f.wait(25 * time.Minute)
	busy = false
	f.wait(time.Second)
	if flushes != 1 {
		t.Fatalf("flushes = %d, want the busy one retried", flushes)
	}
}

func TestStateSurvivesRestart(t *testing.T) {
	f := newFixture(t)

	f.SetTodo("a", "Write report")
	f.start()
	f.wait(25 * time.Minute)
	f.wait(2 * time.Minute)
	f.save()

	f.open()

	if f.phase != phase.Break || f.running || f.Remaining() != 3*time.Minute {
		t.Fatalf("restored: phase=%v running=%v remaining=%v", f.phase, f.running, f.Remaining())
	}
	if f.TodoID() != "a" || f.TodoText() != "Write report" {
		t.Fatalf("restored todo: %q %q", f.TodoID(), f.TodoText())
	}
	if f.SessionsToday() != 1 || f.Stat("a").Sessions != 1 {
		t.Fatalf("restored history: today=%d stat=%+v", f.SessionsToday(), f.Stat("a"))
	}
}

func TestOtherTodosStartTheNextPhaseFresh(t *testing.T) {
	f := newFixture(t)

	f.SetTodo("a", "Write report")
	f.start()
	f.wait(5 * time.Minute)

	f.SetTodo("b", "Review PR")
	f.wait(25 * time.Minute)
	if f.phase != phase.Break {
		t.Fatalf("phase = %v, want the break to have started", f.phase)
	}

	f.SetTodo("a", "Write report")
	if got := f.Remaining(); got != 5*time.Minute {
		t.Fatalf("todo a remaining = %v, want a fresh 5m break", got)
	}
}
