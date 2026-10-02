package timer

import (
	"sync"
	"time"

	"github.com/mt-shihab26/orivo/src/config"
	"github.com/mt-shihab26/orivo/src/systems/logx"
	"github.com/mt-shihab26/orivo/src/systems/phase"
	"github.com/mt-shihab26/orivo/src/systems/sessions"
	"github.com/mt-shihab26/orivo/src/systems/store"
)

type State struct {
	mu     sync.Mutex
	cfg    config.Timer
	store  *store.Store
	log    *sessions.Log
	now    func() time.Time
	notify func(summary, body string)

	running        bool
	remaining      time.Duration
	startedAt      time.Time
	phaseStartedAt time.Time
	phase          phase.Phase
	todoID         string
	todoText       string
	showMillis     bool
}

type Snapshot struct {
	Phase         phase.Phase
	Running       bool
	Remaining     time.Duration
	Total         time.Duration
	ShowMillis    bool
	TodoID        string
	TodoText      string
	Stat          sessions.Stat
	SessionsToday int
	DailyGoal     int
}

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

func (s *State) SetTodo(id, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stash()

	s.todoID, s.todoText = id, text
	s.restore()

	if s.running {
		s.startedAt = s.now()
		if s.phaseStartedAt.IsZero() {
			s.phaseStartedAt = s.now()
			s.store.SetPhaseStartedAt(s.todoID, s.phaseStartedAt)
		}
	}

	s.store.SetTodo(s.todoID, s.todoText)
	s.store.Save()
}

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

func (s *State) Skip() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advance()
}

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

func (s *State) Tick() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running && s.current() == 0 {
		s.advance()
	}
}

func (s *State) Save() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stash()
	s.store.Save()
}

func (s *State) current() time.Duration {
	if s.startedAt.IsZero() {
		return s.remaining
	}
	return max(s.remaining-s.now().Sub(s.startedAt), 0)
}

func (s *State) stash() {
	s.store.SetRemaining(s.todoID, s.current())
	if !s.phaseStartedAt.IsZero() {
		s.store.SetPhaseStartedAt(s.todoID, s.phaseStartedAt)
	}
}

func (s *State) restore() {
	remaining, ok := s.store.Remaining(s.todoID)
	if !ok {
		remaining = s.phase.Duration(s.cfg)
	}
	s.remaining = remaining
	s.phaseStartedAt, _ = s.store.PhaseStartedAt(s.todoID)
}

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

	s.store.ClearRemaining(s.todoID)
	s.store.SetPhase(s.phase)
	s.store.Save()
}

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
