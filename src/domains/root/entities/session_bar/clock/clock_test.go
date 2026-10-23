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

	f.Clock = newClock(cfg.Timer, nil, filepath.Join(f.dir, "store.json"), filepath.Join(f.dir, "sessions"), filepath.Join(f.dir, "todoist.txt"))
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
		t.Fatalf("sessions today = %v, want 1", f.SessionsToday())
	}
	if f.Accent() != phase.Break.Color() {
		t.Fatalf("accent did not follow the phase")
	}

	f.wait(5 * time.Minute)

	if f.phase != phase.Work || f.running || f.Remaining() != 25*time.Minute {
		t.Fatalf("after break: phase=%v running=%v remaining=%v", f.phase, f.running, f.Remaining())
	}
	if f.SessionsToday() != 1 {
		t.Fatalf("a break counted as a session: %v", f.SessionsToday())
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
		t.Fatalf("next day: today=%v stat=%+v, want both zero", f.SessionsToday(), f.Stat("a"))
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
		t.Fatalf("restored history: today=%v stat=%+v", f.SessionsToday(), f.Stat("a"))
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

func TestChangedOnlyWhenTheShownTimeMoves(t *testing.T) {
	f := newFixture(t)
	drawn := func() { f.drawnText = clockText(f.Remaining(), f.showMillis) }

	drawn()
	f.wait(time.Minute)
	if f.Changed() {
		t.Fatal("a paused clock asked to be drawn again")
	}

	f.start()
	f.wait(400 * time.Millisecond)
	if f.Changed() {
		t.Fatal("asked to be drawn again within the same second")
	}
	f.wait(700 * time.Millisecond)
	if !f.Changed() {
		t.Fatal("the next second did not ask to be drawn")
	}

	drawn()
	f.showMillis = true
	if !f.Changed() {
		t.Fatal("showing milliseconds did not ask to be drawn")
	}
}

func TestNextWakesAsTheShownSecondChanges(t *testing.T) {
	f := newFixture(t)
	if got := f.Next(); got != 0 {
		t.Fatalf("paused clock wants a wake in %v", got)
	}

	f.start()
	f.wait(300 * time.Millisecond)
	if got := f.Next(); got != 701*time.Millisecond {
		t.Fatalf("next = %v, want just past the next second", got)
	}

	f.showMillis = true
	if got := f.Next(); got != time.Second/60 {
		t.Fatalf("next = %v, want a frame while showing milliseconds", got)
	}
}

func TestUndoTakesBackTheLastSession(t *testing.T) {
	f := newFixture(t)
	var sent []string
	f.progress = func(todoID, line string) { sent = append(sent, todoID+" "+line) }

	f.SetTodo("a", "Write report")
	f.start()
	f.wait(25 * time.Minute) // work ends, the break starts
	f.wait(time.Minute)

	f.undo(f.time)
	if want := []string{"Work Session Complete", "Work Session Undone"}; !slices.Equal(f.notified, want) {
		t.Fatalf("notified = %q, want %q", f.notified, want)
	}
	if f.phase != phase.Work || f.running || f.Remaining() != 25*time.Minute {
		t.Fatalf("after undo: phase=%v running=%v remaining=%v, want paused work at 25m", f.phase, f.running, f.Remaining())
	}
	if f.SessionsToday() != 0 || f.Stat("a").Sessions != 0 {
		t.Fatalf("sessions = %v, stat = %+v, want none left", f.SessionsToday(), f.Stat("a"))
	}
	if want := []string{"a (1, 25 min)", "a "}; !slices.Equal(sent, want) {
		t.Fatalf("sent = %q, want %q", sent, want)
	}

	f.undo(f.time)
	if f.phase != phase.Work || f.SessionsToday() != 0 {
		t.Fatalf("undo with nothing left changed the clock: phase=%v sessions=%v", f.phase, f.SessionsToday())
	}
}

func TestUndoStepsBackThroughBreaks(t *testing.T) {
	f := newFixture(t)

	f.start()
	f.wait(25 * time.Minute) // work ends
	f.wait(5 * time.Minute)  // the break ends

	f.undo(f.time)
	if f.phase != phase.Break || f.SessionsToday() != 1 {
		t.Fatalf("phase=%v sessions=%v, want the break back with the work kept", f.phase, f.SessionsToday())
	}
	f.undo(f.time)
	if f.phase != phase.Work || f.SessionsToday() != 0 {
		t.Fatalf("phase=%v sessions=%v, want the work back", f.phase, f.SessionsToday())
	}
}

func TestDoneEndsWorkWithTheShareThatRan(t *testing.T) {
	f := newFixture(t)
	var sent, completed []string
	f.progress = func(todoID, line string) { sent = append(sent, todoID+" "+line) }
	f.complete = func(todoID string) { completed = append(completed, todoID) }

	f.SetTodo("a", "Write report")
	f.start()
	f.wait(25 * time.Minute) // a whole session
	f.wait(5 * time.Minute)  // the break
	f.start()
	f.wait(10 * time.Minute)
	f.done(f.time)

	if got := f.SessionsToday(); got != 1.4 {
		t.Fatalf("sessions today = %v, want 1.4", got)
	}
	if stat := f.Stat("a"); stat.Sessions != 1.4 || stat.Secs != 35*60 {
		t.Fatalf("stat = %+v, want 1.4 sessions of 35 min", stat)
	}
	if want := []string{"a (1, 25 min)", "a (1.4, 35 min)"}; !slices.Equal(sent, want) {
		t.Fatalf("sent = %q, want %q", sent, want)
	}
	if !slices.Equal(completed, []string{"a"}) || f.TodoID() != "" {
		t.Fatalf("completed = %q, todo = %q, want a completed and cleared", completed, f.TodoID())
	}
	// The cut-short session still counts towards the long break.
	if f.phase != phase.LongBreak || !f.running {
		t.Fatalf("phase=%v running=%v, want a running long break", f.phase, f.running)
	}
	if f.notified[len(f.notified)-1] != "Todo Done" {
		t.Fatalf("notified = %q", f.notified)
	}
}

func TestDoneBeforeWorkStartsRecordsNothing(t *testing.T) {
	f := newFixture(t)
	var completed []string
	f.complete = func(todoID string) { completed = append(completed, todoID) }

	f.SetTodo("a", "Write report")
	f.done(f.time)

	if f.SessionsToday() != 0 || f.phase != phase.Work || f.Remaining() != 25*time.Minute {
		t.Fatalf("sessions=%v phase=%v remaining=%v, want work untouched", f.SessionsToday(), f.phase, f.Remaining())
	}
	if !slices.Equal(completed, []string{"a"}) || f.TodoID() != "" {
		t.Fatalf("completed = %q, todo = %q", completed, f.TodoID())
	}
}

func TestDoneInABreakOnlyCompletesTheTodo(t *testing.T) {
	f := newFixture(t)
	var completed []string
	f.complete = func(todoID string) { completed = append(completed, todoID) }

	f.SetTodo("a", "Write report")
	f.start()
	f.wait(25 * time.Minute)
	f.wait(time.Minute)
	f.done(f.time)

	if f.SessionsToday() != 1 || f.phase != phase.Break {
		t.Fatalf("sessions=%v phase=%v, want the break kept", f.SessionsToday(), f.phase)
	}
	if !slices.Equal(completed, []string{"a"}) {
		t.Fatalf("completed = %q", completed)
	}
}

func TestDoneWithoutATodoStillEndsWorkEarly(t *testing.T) {
	f := newFixture(t)
	completed := 0
	f.complete = func(string) { completed++ }

	f.start()
	f.wait(5 * time.Minute)
	f.done(f.time)

	if f.SessionsToday() != 0.2 || f.phase != phase.Break || completed != 0 {
		t.Fatalf("sessions=%v phase=%v completed=%d", f.SessionsToday(), f.phase, completed)
	}
}
