use sea_orm::DatabaseConnection;

use crate::models::{
    session::{Session, Stat},
    todo::Todo,
};

/// Per-tab cache for today's todos and their session stats.
pub struct TimerCache {
    db: DatabaseConnection,
    /// Cached list of overdue todos, `None` until first fetch.
    due_todos: Option<Vec<Todo>>,
    /// Cached stats parallel to `due_todos`, `None` until first fetch.
    due_stats: Option<Vec<Stat>>,
    /// Cached list of today's todos, `None` until first fetch.
    todos: Option<Vec<Todo>>,
    /// Cached stats parallel to `todos`, `None` until first fetch.
    stats: Option<Vec<Stat>>,
    /// Cached count of today's completed work sessions, `None` until first fetch.
    today_sessions: Option<u32>,
}

impl TimerCache {
    pub fn new(db: DatabaseConnection) -> Self {
        Self {
            db,
            due_todos: None,
            due_stats: None,
            todos: None,
            stats: None,
            today_sessions: None,
        }
    }

    /// Returns the cached overdue todo list, querying the DB if needed.
    pub fn get_due_todos(&mut self) -> &[Todo] {
        if self.due_todos.is_none() {
            self.due_todos = Some(Todo::list_due_pending(&self.db));
        }
        self.due_todos.as_deref().unwrap_or(&[])
    }

    /// Returns stats for all cached overdue todos, querying the DB if needed.
    pub fn get_due_stats(&mut self) -> &[Stat] {
        if self.due_stats.is_none() {
            let ids: Vec<Option<i32>> = self.get_due_todos().iter().map(|t| t.id).collect();
            let stats = ids
                .into_iter()
                .map(|id| {
                    id.map(|id| Session::stat(&self.db, id))
                        .unwrap_or(Stat::new(0, 0))
                })
                .collect();
            self.due_stats = Some(stats);
        }
        self.due_stats.as_deref().unwrap_or(&[])
    }

    /// Returns the cached todo list (excluding done todos), querying the DB if needed.
    pub fn get_todos(&mut self) -> &[Todo] {
        if self.todos.is_none() {
            self.todos = Some(Todo::list_today_pending(&self.db));
        }
        self.todos.as_deref().unwrap_or(&[])
    }

    /// Returns the cached todo with the given id, querying the DB if needed.
    pub fn get_todo(&mut self, id: i32) -> Option<&Todo> {
        self.get_due_todos();
        self.get_todos();
        self.due_todos
            .as_deref()?
            .iter()
            .chain(self.todos.as_deref().unwrap_or(&[]).iter())
            .find(|t| t.id == Some(id))
    }

    /// Returns stats for all cached todos, querying the DB if needed.
    pub fn get_stats(&mut self) -> &[Stat] {
        if self.stats.is_none() {
            let ids: Vec<Option<i32>> = self.get_todos().iter().map(|t| t.id).collect();
            let stats = ids
                .into_iter()
                .map(|id| {
                    id.map(|id| Session::stat(&self.db, id))
                        .unwrap_or(Stat::new(0, 0))
                })
                .collect();
            self.stats = Some(stats);
        }
        self.stats.as_deref().unwrap_or(&[])
    }

    /// Returns the cached session stats for the given todo id, querying the DB if needed.
    pub fn get_stat(&mut self, todo_id: i32) -> Option<&Stat> {
        self.get_due_stats();
        self.get_stats();
        if let Some(idx) = self
            .due_todos
            .as_deref()?
            .iter()
            .position(|t| t.id == Some(todo_id))
        {
            return self.due_stats.as_deref()?.get(idx);
        }
        let idx = self
            .todos
            .as_deref()?
            .iter()
            .position(|t| t.id == Some(todo_id))?;
        self.stats.as_deref()?.get(idx)
    }

    /// Returns the number of today's completed work sessions, querying the DB if needed.
    pub fn get_today_sessions(&mut self) -> u32 {
        if self.today_sessions.is_none() {
            self.today_sessions = Some(Session::count_today(&self.db));
        }
        self.today_sessions.unwrap_or(0)
    }

    /// Drops the cached todo list so the next call to `get_todos()` re-queries.
    pub fn invalidate_todos(&mut self) {
        self.due_todos = None;
        self.due_stats = None;
        self.todos = None;
        self.stats = None;
        self.today_sessions = None;
    }

    /// Drops the cached session stats so they are re-fetched on next render.
    pub fn invalidate_stats(&mut self) {
        self.due_stats = None;
        self.stats = None;
        self.today_sessions = None;
    }
}
