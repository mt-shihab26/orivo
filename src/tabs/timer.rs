use std::{
    io::Result,
    sync::{
        Arc, Mutex,
        atomic::{AtomicU8, Ordering},
        mpsc::Sender,
    },
};

use sea_orm::DatabaseConnection;

use ratatui::{
    Frame,
    crossterm::event::{KeyCode, KeyEvent},
    layout::{Constraint, Layout, Rect},
    style::Color,
    widgets::Widget,
};

use crate::{
    caches::timer::TimerCache,
    config::timer::TimerConfig,
    kinds::{event::Event, phase::Phase},
    log_warn,
    models::{session::Stat, todo::Todo},
    states::timer::TimerState,
    utils::store::Store,
    widgets::{
        layout::border::{BorderProps, BorderWidget},
        timer::{
            clock::{ClockProps, ClockWidget},
            hint::{HintProps, HintWidget},
            phase::{PhaseProps, PhaseWidget},
            reduce_picker::{
                ReducePickerAction, ReducePickerProps, ReducePickerState, ReducePickerWidget,
            },
            session::{SessionProps, SessionWidget},
            status::{StatusProps, StatusWidget},
            todo_picker::{TodoPickerAction, TodoPickerProps, TodoPickerState, TodoPickerWidget},
            todo_show::{TodoShowProps, TodoShowWidget},
        },
    },
    workers::{ipc, timer::spawn},
};

use super::Tab;

/// The timer tab, managing the pomodoro timer UI and its worker thread.
pub struct TimerTab {
    /// Render counter shared with the worker thread to pace `TimerTick` events.
    count: Arc<AtomicU8>,
    /// Shared cache for today's todos and session stats.
    cache: Arc<Mutex<TimerCache>>,
    /// Shared timer state owned by the worker thread.
    state: Arc<Mutex<TimerState>>,
    /// Active todo picker overlay, `None` when closed.
    picker: Option<TodoPickerState>,
    /// Active reduce-time dialog, `None` when closed.
    reduce_state: Option<ReducePickerState>,
}

impl TimerTab {
    /// Creates a new `TimerTab`, spawning the timer worker thread.
    pub fn new(
        sender: Sender<Event>,
        db: DatabaseConnection,
        config: TimerConfig,
        cache: Arc<Mutex<TimerCache>>,
        store: Store,
    ) -> Self {
        let count = Arc::new(AtomicU8::new(1));
        let state = spawn(
            Arc::clone(&count),
            sender,
            db,
            config,
            Arc::clone(&cache),
            store,
        );

        ipc::spawn(Arc::clone(&state));

        Self {
            count,
            cache,
            state,
            picker: None,
            reduce_state: None,
        }
    }

    /// Sets the currently associated todo on the timer state.
    fn set_todo(&mut self, todo_id: Option<i32>) {
        if let Ok(mut state) = self.state.lock() {
            state.set_todo_id(todo_id);
        }
    }

    /// Confirms the picker selection and closes the overlay.
    fn picker_select(&mut self, id: i32) {
        self.set_todo(Some(id));
        self.picker = None;
    }

    /// Closes the picker without changing the selected todo.
    fn picker_cancel(&mut self) {
        self.picker = None;
    }

    /// Toggles the timer between running and paused.
    fn toggle_running(&self) {
        if let Ok(mut state) = self.state.lock() {
            state.toggle_running();
        }
    }

    /// Resets the timer to the full duration of the current phase without advancing.
    fn reset_timer(&self) {
        if let Ok(mut state) = self.state.lock() {
            state.reset();
        }
    }

    /// Records the current session and advances to the next phase.
    fn skip_session(&self) {
        if let Ok(mut state) = self.state.lock() {
            state.advance();
        }
    }

    /// Opens the todo picker, loading todos and stats from the cache.
    fn open_picker(&mut self) {
        if let Ok(mut cache) = self.cache.lock() {
            let due_todos = cache.get_due_todos().to_vec();
            let due_stats = cache.get_due_stats().to_vec();
            let todos = cache.get_todos().to_vec();
            let stats = cache.get_stats().to_vec();
            let selected_id = self.state.lock().ok().and_then(|s| s.todo_id());
            self.picker = Some(TodoPickerState::new(TodoPickerProps::new(
                due_todos,
                due_stats,
                todos,
                stats,
                self.color(),
                selected_id,
            )));
        }
    }

    /// Clears the currently associated todo from the timer state.
    fn clear_todo(&mut self) {
        self.set_todo(None);
    }

    /// Opens the reduce-time dialog.
    fn open_reduce_picker(&mut self) {
        self.reduce_state = Some(ReducePickerState::new(ReducePickerProps::new(self.color())));
    }

    /// Applies the reduction and closes the dialog.
    fn reduce_picker_apply(&mut self, millis: u32) {
        if let Ok(mut state) = self.state.lock() {
            state.reduce_remaining(millis);
        }
        self.reduce_state = None;
    }

    /// Closes the reduce dialog without applying.
    fn reduce_picker_cancel(&mut self) {
        self.reduce_state = None;
    }

    /// Toggles millisecond display on the clock.
    fn toggle_millis(&self) {
        if let Ok(mut state) = self.state.lock() {
            state.toggle_show_millis();
        }
    }

    /// Increments the render counter so the worker thread knows a new frame was drawn.
    fn tick_render_count(&self) {
        self.count.fetch_add(1, Ordering::Relaxed);
    }

    /// Returns the todo and stat for the currently selected todo id, if any.
    fn todo_info(&self) -> (Option<Todo>, Option<Stat>) {
        let Some(id) = self.state.lock().ok().and_then(|state| state.todo_id()) else {
            return (None, None);
        };
        let Ok(mut cache) = self.cache.lock() else {
            return (None, None);
        };
        let todo = cache.get_todo(id).cloned();
        let stat = cache.get_stat(id).cloned();
        (todo, stat)
    }
}

impl Drop for TimerTab {
    fn drop(&mut self) {
        if let Ok(mut state) = self.state.lock() {
            state.save_remaining();
        }
    }
}

impl Tab for TimerTab {
    /// Returns the tab label shown in the tab bar.
    fn name(&self) -> &str {
        "Timer [^y]"
    }

    /// Returns the accent color for the timer tab.
    fn color(&self) -> Color {
        if let Ok(state) = self.state.lock() {
            state.cycle_phase().color()
        } else {
            Phase::Work.color()
        }
    }

    /// Handles a key event, delegating to the active overlay or timer controls.
    fn handle(&mut self, key: KeyEvent) -> Result<()> {
        if let Some(picker) = &mut self.picker {
            match picker.handle(key) {
                TodoPickerAction::Select(id) => self.picker_select(id),
                TodoPickerAction::Cancel => self.picker_cancel(),
                TodoPickerAction::None => {}
            }
            return Ok(());
        }

        if let Some(reduce) = &mut self.reduce_state {
            match reduce.handle(key) {
                ReducePickerAction::Reduce(millis) => self.reduce_picker_apply(millis),
                ReducePickerAction::Cancel => self.reduce_picker_cancel(),
                ReducePickerAction::None => {}
            }
            return Ok(());
        }

        match key.code {
            KeyCode::Char(' ') => self.toggle_running(),
            KeyCode::Char('r') => self.reset_timer(),
            KeyCode::Char('n') => self.skip_session(),
            KeyCode::Char('t') => self.open_picker(),
            KeyCode::Char('T') => self.clear_todo(),
            KeyCode::Char('m') => self.toggle_millis(),
            KeyCode::Char('d') => self.open_reduce_picker(),
            _ => {}
        }

        Ok(())
    }

    /// Renders the timer tab including clock, phase, session, status, todo bar, and hints.
    fn render(&self, frame: &mut Frame, area: Rect) {
        self.tick_render_count();

        let state = match self.state.lock() {
            Ok(guard) => guard,
            Err(poisoned) => {
                log_warn!("timer state mutex poisoned in render, recovering");
                poisoned.into_inner()
            }
        };

        let cycle_phase = state.cycle_phase();

        let phase_color = cycle_phase.color();
        let phase_label = cycle_phase.label().to_string();

        let sessions = state.sessions_count();
        let running = state.is_running();
        let daily_session_goal = state.daily_session_goal();
        let show_millis = state.show_millis();
        let time_millis = state.current_millis();

        drop(state);

        let buf = frame.buffer_mut();

        let inner = BorderWidget::new(&BorderProps::new(self.color()), area).render(area, buf);

        let [
            session_row,
            _,
            status_row,
            phase_row,
            _,
            time_row,
            _,
            todo_row,
            _,
            hint_row,
        ] = Layout::vertical([
            Constraint::Length(2),
            Constraint::Fill(1),
            Constraint::Length(1),
            Constraint::Length(1),
            Constraint::Length(1),
            Constraint::Length(8),
            Constraint::Length(1),
            Constraint::Length(1),
            Constraint::Fill(1),
            Constraint::Length(1),
        ])
        .areas(inner);

        SessionWidget::new(&SessionProps::new(
            sessions,
            daily_session_goal,
            phase_color,
        ))
        .render(session_row, buf);
        PhaseWidget::new(&PhaseProps::new(phase_label, phase_color)).render(phase_row, buf);
        ClockWidget::new(&ClockProps::new(show_millis, time_millis, phase_color))
            .render(time_row, buf);
        StatusWidget::new(&StatusProps::new(running, phase_color)).render(status_row, buf);

        let (todo, stat) = self.todo_info();

        TodoShowWidget::new(&TodoShowProps::new(
            todo.as_ref(),
            stat.as_ref(),
            phase_color,
        ))
        .render(todo_row, buf);
        HintWidget::new(&HintProps::new(
            self.picker.is_some(),
            self.reduce_state.is_some(),
        ))
        .render(hint_row, buf);

        if let Some(picker) = &self.picker {
            TodoPickerWidget::new(&picker.props()).render(inner, buf);
        }

        if let Some(reduce) = &self.reduce_state {
            ReducePickerWidget::new(reduce.props()).render(inner, buf);
        }
    }

    /// Drops any cached data held by this tab.
    fn invalidate_cache(&mut self) {
        if let Ok(mut cache) = self.cache.lock() {
            cache.invalidate_todos();
        }
    }
}
