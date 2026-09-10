use std::{
    sync::{Arc, Mutex},
    time::Instant,
};

use sea_orm::DatabaseConnection;
use time::OffsetDateTime;

use crate::{
    caches::timer::TimerCache,
    config::timer::TimerConfig,
    kinds::phase::Phase,
    models::session::Session,
    utils::{date::now, notify::notify, store::Store},
};

/// Runtime state for the pomodoro timer, owned by the timer worker thread.
pub struct TimerState {
    /// Database connection used to persist sessions.
    db: DatabaseConnection,
    /// Timer configuration (durations, intervals, display options).
    config: TimerConfig,
    /// Shared cache reference, used to invalidate stats after a session completes.
    cache: Arc<Mutex<TimerCache>>,
    /// Whether the timer is actively counting down.
    is_running: bool,
    /// Remaining time captured at the last pause or resume.
    remaining_millis: u32,
    /// Wall-clock anchor set when the timer was last resumed, `None` when paused.
    started_at: Option<Instant>,
    /// UTC timestamp of when the current phase was first started, `None` before first resume.
    phase_started_at: Option<OffsetDateTime>,
    /// Current phase of the pomodoro cycle (work, break, or long break).
    cycle_phase: Phase,
    /// The currently selected todo id, used to associate sessions.
    todo_id: Option<i32>,
    /// Whether to display milliseconds on the clock, toggleable at runtime.
    show_millis: bool,
    /// Persisted state, synced to disk on change.
    store: Store,
}

impl TimerState {
    /// Creates a new `TimerState` in the initial paused work phase.
    pub fn new(
        db: DatabaseConnection,
        config: TimerConfig,
        cache: Arc<Mutex<TimerCache>>,
        store: Store,
    ) -> Self {
        let show_millis = config.show_millis();

        let todo_id = store.timer_todo_id();

        let cycle_phase = store.timer_cycle_phase().clone();

        let remaining_millis = store
            .timer_remaining_for_todo(todo_id)
            .unwrap_or_else(|| cycle_phase.duration(&config));

        let phase_started_at = store.timer_phase_started_at_for_todo(todo_id);

        Self {
            cycle_phase,
            remaining_millis,
            started_at: None,
            phase_started_at,
            is_running: false,
            config,
            db,
            todo_id,
            cache,
            show_millis,
            store,
        }
    }

    /// Returns the currently associated todo id, if any.
    pub fn todo_id(&self) -> Option<i32> {
        self.todo_id
    }

    /// Returns the current phase of the pomodoro cycle.
    pub fn cycle_phase(&self) -> &Phase {
        &self.cycle_phase
    }

    /// Returns the text of the currently associated todo, if any.
    pub fn todo_text(&self) -> Option<String> {
        let id = self.todo_id?;
        let mut cache = self.cache.lock().ok()?;
        cache.get_todo(id).map(|t| t.text.clone())
    }

    /// Returns the number of completed work sessions today, loaded from the cache.
    pub fn sessions_count(&self) -> u32 {
        if let Ok(mut c) = self.cache.lock() {
            c.get_today_sessions()
        } else {
            0
        }
    }

    /// Returns whether the timer is actively counting down.
    pub fn is_running(&self) -> bool {
        self.is_running
    }

    /// Returns whether milliseconds are shown on the clock.
    pub fn show_millis(&self) -> bool {
        self.show_millis
    }

    /// Returns the number of work sessions before a long break.
    pub fn long_break_interval(&self) -> u32 {
        self.config.long_break_interval()
    }

    /// Returns the configured daily session goal.
    pub fn daily_session_goal(&self) -> u32 {
        self.config.daily_session_goal()
    }

    /// Sets the currently associated todo on the timer state and persists it to disk.
    pub fn set_todo_id(&mut self, todo_id: Option<i32>) {
        // Persist remaining time and phase start for the outgoing todo before switching.
        let current = self.current_millis();
        self.store
            .set_timer_remaining_for_todo(self.todo_id, current);
        if let Some(t) = self.phase_started_at {
            self.store
                .set_timer_phase_started_at_for_todo(self.todo_id, t);
        }

        self.todo_id = todo_id;

        // Restore saved remaining and phase start for the incoming todo.
        self.remaining_millis = self
            .store
            .timer_remaining_for_todo(self.todo_id)
            .unwrap_or_else(|| self.cycle_phase.duration(&self.config));
        self.phase_started_at = self.store.timer_phase_started_at_for_todo(self.todo_id);

        // Re-anchor the wall-clock start so the restored time is used correctly.
        if self.is_running {
            self.started_at = Some(Instant::now());
            // If the timer is running but the incoming todo has no recorded phase start,
            // stamp now so advance() records a sane session start instead of falling
            // back to the completion time.
            if self.phase_started_at.is_none() {
                let started = now();
                self.phase_started_at = Some(started);
                self.store
                    .set_timer_phase_started_at_for_todo(self.todo_id, started);
            }
        }

        self.store.set_timer_todo_id(self.todo_id).save();
    }

    /// Subtracts `millis` from the current remaining time and persists the result.
    pub fn reduce_remaining(&mut self, millis: u32) {
        let new_remaining = self.current_millis().saturating_sub(millis);
        self.remaining_millis = new_remaining;
        if self.is_running {
            self.started_at = Some(Instant::now());
        }
        self.store
            .set_timer_remaining_for_todo(self.todo_id, new_remaining)
            .save();
    }

    /// Saves the current remaining time and phase start for the active todo to the store.
    pub fn save_remaining(&mut self) {
        let millis = self.current_millis();
        self.store
            .set_timer_remaining_for_todo(self.todo_id, millis);
        if let Some(t) = self.phase_started_at {
            self.store
                .set_timer_phase_started_at_for_todo(self.todo_id, t);
        }
        self.store.save();
    }

    /// Returns the current remaining time derived from the wall clock.
    pub fn current_millis(&self) -> u32 {
        match self.started_at {
            Some(t) => self
                .remaining_millis
                .saturating_sub(t.elapsed().as_millis() as u32),
            None => self.remaining_millis,
        }
    }

    /// Toggles between running and paused, anchoring to the wall clock on resume.
    pub fn toggle_running(&mut self) {
        if self.is_running {
            self.remaining_millis = self.current_millis();
            self.started_at = None;
            self.is_running = false;
            self.store
                .set_timer_remaining_for_todo(self.todo_id, self.remaining_millis)
                .save();
        } else {
            if self.phase_started_at.is_none() {
                let started = now();
                self.phase_started_at = Some(started);
                self.store
                    .set_timer_phase_started_at_for_todo(self.todo_id, started);
                self.store.save();
            }
            self.started_at = Some(Instant::now());
            self.is_running = true;
        }
    }

    /// Toggles whether milliseconds are shown on the clock.
    pub fn toggle_show_millis(&mut self) {
        self.show_millis = !self.show_millis;
    }

    /// Resets the timer to the full duration of the current phase without advancing.
    pub fn reset(&mut self) {
        self.remaining_millis = self.cycle_phase.duration(&self.config);
        self.started_at = None;
        self.phase_started_at = None;
        self.is_running = false;
        self.store.clear_timer_remaining_for_todo(self.todo_id);
        self.store
            .clear_timer_phase_started_at_for_todo(self.todo_id);
        self.store.save();
    }

    /// Called every tick — advances the phase if time has expired.
    pub fn tick(&mut self) {
        if self.is_running && self.current_millis() == 0 {
            self.advance();
        }
    }

    /// Records the current session and moves to the next phase.
    pub fn advance(&mut self) {
        self.phase_notification();

        // `take()` clears phase_started_at in memory; clear the store entry too.
        let phase_started_at = self.phase_started_at.take().unwrap_or_else(now);
        self.store
            .clear_timer_phase_started_at_for_todo(self.todo_id);

        Session::record(
            &self.db,
            &self.cycle_phase,
            self.cycle_phase.duration(&self.config),
            phase_started_at,
            self.todo_id,
        );

        match self.cycle_phase {
            Phase::Work => {
                let sessions_count = if let Ok(mut c) = self.cache.lock() {
                    c.invalidate_stats();
                    c.get_today_sessions()
                } else {
                    1
                };
                self.cycle_phase = self.which_break_phase(sessions_count);
                self.is_running = true;
            }
            Phase::Break | Phase::LongBreak => {
                self.cycle_phase = Phase::Work;
                self.is_running = false;
            }
        }

        self.remaining_millis = self.cycle_phase.duration(&self.config);
        self.started_at = if self.is_running {
            Some(Instant::now())
        } else {
            None
        };
        self.phase_started_at = if self.is_running { Some(now()) } else { None };

        // Persist the new phase start if the next phase auto-started (e.g. after Work completes).
        if let Some(t) = self.phase_started_at {
            self.store
                .set_timer_phase_started_at_for_todo(self.todo_id, t);
        }
        // Clear saved remaining — the phase just changed, so the full duration applies.
        self.store.clear_timer_remaining_for_todo(self.todo_id);
        self.store
            .set_timer_cycle_phase(self.cycle_phase.clone())
            .save();
    }

    /// Sends a desktop notification describing the completed phase.
    fn phase_notification(&self) {
        let todo_name = self
            .todo_id
            .and_then(|id| self.cache.lock().ok()?.get_todo(id).map(|t| t.text.clone()));

        let (summary, body) = match (&self.cycle_phase, todo_name.as_deref()) {
            (Phase::Work, Some(name)) => (
                "Work Session Complete",
                format!("{name} — Time for a break!"),
            ),
            (Phase::Work, None) => ("Work Session Complete", "Time for a break!".to_string()),
            (Phase::Break, Some(name)) => ("Break Complete", format!("{name} — Ready to focus?")),
            (Phase::Break, None) => ("Break Complete", "Ready to focus?".to_string()),
            (Phase::LongBreak, Some(name)) => {
                ("Long Break Complete", format!("{name} — Ready to focus?"))
            }
            (Phase::LongBreak, None) => ("Long Break Complete", "Ready to focus?".to_string()),
        };
        notify(&summary, &body, &self.cycle_phase);
    }

    /// Chooses whether the next rest phase is a short break or a long break.
    fn which_break_phase(&self, sessions_count: u32) -> Phase {
        if sessions_count % self.config.long_break_interval() == 0 {
            Phase::LongBreak
        } else {
            Phase::Break
        }
    }
}
