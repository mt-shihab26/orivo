// Package timer holds the pomodoro state machine.
package timer

import (
	"sync"
	"time"

	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/logx"
	"github.com/mt-shihab26/orivo/src/phase"
	"github.com/mt-shihab26/orivo/src/sessions"
	"github.com/mt-shihab26/orivo/src/store"
)

// State is the runtime state of the timer. It is safe for concurrent use: the
// window, the background ticker and the IPC server all share one.
type State struct {
	mu    sync.Mutex
	cfg   config.Timer
	store *store.Store
	log   *sessions.Log
	now   func() time.Time
	// Called with a summary and body whenever a phase ends. It runs with the
	// state locked, so it must not call back into the state.
	notify func(summary, body string)

	running bool
	// Remaining time captured at the last pause or resume.
	remaining time.Duration
	// Anchor set when the timer was last resumed, zero when paused.
	startedAt time.Time
	// When the current phase was first started, zero before the first resume.
	phaseStartedAt time.Time
	phase          phase.Phase
	// The selected todo, which sessions are recorded against; empty when none.
	todoID   string
	todoText string
	// Whether the clock shows centiseconds, toggleable at runtime.
	showMillis bool
}

// Snapshot is everything needed to present the timer at one instant.
type Snapshot struct {
	Phase     phase.Phase
	Running   bool
	Remaining time.Duration
	// Total is the full length of the current phase.
	Total      time.Duration
	ShowMillis bool
	TodoID     string
	TodoText   string
	// Stat covers the selected todo; zero when none is selected.
	Stat          sessions.Stat
	SessionsToday int
	DailyGoal     int
}

// New restores the timer from the store, paused.
func New(cfg config.Timer, st *store.Store, log *sessions.Log) *State {
	s := &State{
		cfg:        cfg,
		store:      st,
		log:        log,
		now:        time.Now,
		phase:      st.Phase(),
		showMillis: cfg.ShowMillis,
	}
	s.todoID, s.todoText = st.Todo()
	s.restore()
	return s
}

// OnPhaseEnd sets the function called whenever a phase ends; nil disables it.
func (s *State) OnPhaseEnd(fn func(summary, body string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notify = fn
}

func (s *State) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap := Snapshot{
		Phase:         s.phase,
		Running:       s.running,
		Remaining:     s.current(),
		Total:         s.phase.Duration(s.cfg),
		ShowMillis:    s.showMillis,
		TodoID:        s.todoID,
		TodoText:      s.todoText,
		SessionsToday: s.log.CountToday(s.now()),
		DailyGoal:     s.cfg.Goal(),
	}
	if s.todoID != "" {
		snap.Stat = s.log.Stat(s.todoID)
	}
	return snap
}

// SetTodo points the timer at a todo, or at none when id is empty. Each todo
// keeps its own remaining time, which is swapped in.
func (s *State) SetTodo(id, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Persist the outgoing todo's progress before switching.
	s.stash()

	s.todoID, s.todoText = id, text
	s.restore()

	// Re-anchor so the restored time counts down from now.
	if s.running {
		s.startedAt = s.now()
		// A todo with no recorded phase start gets one now, so the session is
		// not logged as starting at the moment it completes.
		if s.phaseStartedAt.IsZero() {
			s.phaseStartedAt = s.now()
			s.store.SetPhaseStartedAt(s.todoID, s.phaseStartedAt)
		}
	}

	s.store.SetTodo(s.todoID, s.todoText)
	s.store.Save()
}

// Toggle switches between running and paused.
func (s *State) Toggle() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.remaining = s.current()
		s.startedAt = time.Time{}
		s.running = false
		s.store.SetRemaining(s.todoID, s.remaining)
		s.store.Save()
		return
	}

	if s.phaseStartedAt.IsZero() {
		s.phaseStartedAt = s.now()
		s.store.SetPhaseStartedAt(s.todoID, s.phaseStartedAt)
		s.store.Save()
	}
	s.startedAt = s.now()
	s.running = true
}

// Reset returns the timer to the full length of the current phase, paused.
func (s *State) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.remaining = s.phase.Duration(s.cfg)
	s.startedAt = time.Time{}
	s.phaseStartedAt = time.Time{}
	s.running = false
	s.store.ClearRemaining(s.todoID)
	s.store.ClearPhaseStartedAt(s.todoID)
	s.store.Save()
}

// Skip ends the current phase now, recording it as a completed session.
func (s *State) Skip() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advance()
}

// Reduce subtracts d from the remaining time.
func (s *State) Reduce(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.remaining = max(s.current()-d, 0)
	if s.running {
		s.startedAt = s.now()
	}
	s.store.SetRemaining(s.todoID, s.remaining)
	s.store.Save()
}

func (s *State) ToggleMillis() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.showMillis = !s.showMillis
}

// Tick advances to the next phase if the current one has run out.
func (s *State) Tick() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running && s.current() == 0 {
		s.advance()
	}
}

// Save persists the remaining time of the selected todo.
func (s *State) Save() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stash()
	s.store.Save()
}

// Run ticks the timer until stop is closed, so phases end on time even while
// the window is not being drawn, and saves the remaining time every minute.
func (s *State) Run(stop <-chan struct{}) {
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	save := time.NewTicker(time.Minute)
	defer save.Stop()

	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			s.Tick()
		case <-save.C:
			s.Save()
		}
	}
}

// current is the remaining time derived from the clock.
func (s *State) current() time.Duration {
	if s.startedAt.IsZero() {
		return s.remaining
	}
	return max(s.remaining-s.now().Sub(s.startedAt), 0)
}

// stash writes the selected todo's remaining time and phase start to the store.
func (s *State) stash() {
	s.store.SetRemaining(s.todoID, s.current())
	if !s.phaseStartedAt.IsZero() {
		s.store.SetPhaseStartedAt(s.todoID, s.phaseStartedAt)
	}
}

// restore loads the selected todo's remaining time and phase start from the store.
func (s *State) restore() {
	remaining, ok := s.store.Remaining(s.todoID)
	if !ok {
		remaining = s.phase.Duration(s.cfg)
	}
	s.remaining = remaining
	s.phaseStartedAt, _ = s.store.PhaseStartedAt(s.todoID)
}

// advance records the current phase as a session and moves to the next one.
func (s *State) advance() {
	s.announce()

	now := s.now()
	started := s.phaseStartedAt
	if started.IsZero() {
		started = now
	}
	s.store.ClearPhaseStartedAt(s.todoID)

	err := s.log.Record(sessions.Session{
		Phase:        s.phase.Key(),
		DurationSecs: int(s.phase.Duration(s.cfg).Seconds()),
		StartedAt:    started,
		EndedAt:      now,
		TodoID:       s.todoID,
		TodoText:     s.todoText,
	})
	if err != nil {
		logx.Error("failed to record session: %v", err)
	}

	// A break starts by itself; getting back to work is a deliberate act.
	if s.phase == phase.Work {
		s.phase = s.breakAfter(s.log.CountToday(now))
		s.running = true
	} else {
		s.phase = phase.Work
		s.running = false
	}

	s.remaining = s.phase.Duration(s.cfg)
	s.startedAt = time.Time{}
	s.phaseStartedAt = time.Time{}
	if s.running {
		s.startedAt = now
		s.phaseStartedAt = now
		s.store.SetPhaseStartedAt(s.todoID, now)
	}

	// The phase just changed, so the full duration applies.
	s.store.ClearRemaining(s.todoID)
	s.store.SetPhase(s.phase)
	s.store.Save()
}

// breakAfter picks the rest that follows the given number of work sessions.
func (s *State) breakAfter(sessionsToday int) phase.Phase {
	if sessionsToday%s.cfg.Interval() == 0 {
		return phase.LongBreak
	}
	return phase.Break
}

func (s *State) announce() {
	if s.notify == nil {
		return
	}

	summary, body := "Work Session Complete", "Time for a break!"
	switch s.phase {
	case phase.Break:
		summary, body = "Break Complete", "Ready to focus?"
	case phase.LongBreak:
		summary, body = "Long Break Complete", "Ready to focus?"
	}
	if s.todoText != "" {
		body = s.todoText + " — " + body
	}
	s.notify(summary, body)
}
