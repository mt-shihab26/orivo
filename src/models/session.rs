use std::io;

use sea_orm::{
    ActiveModelBehavior, ActiveModelTrait, ActiveValue::Set, ColumnTrait, DatabaseConnection,
    DeriveEntityModel, DerivePrimaryKey, DeriveRelation, EntityTrait, EnumIter, PrimaryKeyTrait,
    QueryFilter, QueryOrder,
};
use time::OffsetDateTime;

use crate::{
    kinds::phase::Phase,
    log_error,
    utils::{
        date::{format_date, format_datetime, now, parse_datetime, today},
        db::rt,
    },
};

/// Aggregated work session statistics for a single todo.
#[derive(Clone)]
pub struct Stat {
    pub completed_sessions: u32,
    /// Total time spent in seconds across all completed work sessions.
    pub completed_secs: u32,
}

impl Stat {
    pub fn new(completed_sessions: u32, completed_secs: u32) -> Self {
        Self {
            completed_sessions,
            completed_secs,
        }
    }
}

#[derive(Clone)]
pub struct Session {
    /// Database primary key, `None` before the record is saved.
    pub id: Option<i32>,
    pub phase: Phase,
    /// Duration of the session in seconds.
    pub duration_secs: u32,
    /// Local timestamp of when the session started.
    pub started_at: OffsetDateTime,
    /// Local timestamp of when the session ended.
    pub ended_at: OffsetDateTime,
    pub todo_id: Option<i32>,
    pub created_at: OffsetDateTime,
    pub updated_at: OffsetDateTime,
}

impl Session {
    pub fn new(
        phase: &Phase,
        duration_millis: u32,
        started_at: OffsetDateTime,
        todo_id: Option<i32>,
    ) -> Self {
        let now = now();

        Self {
            id: None,
            phase: phase.clone(),
            duration_secs: duration_millis / 1000,
            started_at,
            ended_at: now,
            todo_id,
            created_at: now,
            updated_at: now,
        }
    }

    /// Creates and persists a completed session to the database.
    pub fn record(
        db: &DatabaseConnection,
        phase: &Phase,
        duration_millis: u32,
        started_at: OffsetDateTime,
        todo_id: Option<i32>,
    ) {
        Self::new(phase, duration_millis, started_at, todo_id).save(db);
    }

    fn save(&mut self, db: &DatabaseConnection) -> bool {
        match rt().block_on(async { self.to_model().insert(db).await.map_err(io_err) }) {
            Ok(model) => {
                *self = model.into();
                true
            }
            Err(e) => {
                log_error!("failed to save session: {e}");
                false
            }
        }
    }

    /// Returns the number of completed work sessions started today.
    pub fn count_today(db: &DatabaseConnection) -> u32 {
        let prefix = format!("{}%", format_date(today()));
        match rt().block_on(async {
            Entity::find()
                .filter(Column::Phase.eq(Phase::Work.to_db_str()))
                .filter(Column::EndedAt.like(prefix))
                .all(db)
                .await
                .map_err(io_err)
        }) {
            Ok(rows) => rows.len() as u32,
            Err(e) => {
                log_error!("failed to count today's sessions: {e}");
                0
            }
        }
    }

    /// Returns aggregated work session stats for the given todo.
    pub fn stat(db: &DatabaseConnection, todo_id: i32) -> Stat {
        let sessions: Vec<_> = Self::get(db, todo_id)
            .into_iter()
            .filter(|s| s.phase.to_db_str() == Phase::Work.to_db_str())
            .collect();

        let completed_sessions = sessions.len() as u32;
        let completed_secs = sessions.iter().map(|s| s.duration_secs).sum();

        Stat::new(completed_sessions, completed_secs)
    }

    /// Fetches all sessions for the given todo from the database.
    fn get(db: &DatabaseConnection, todo_id: i32) -> Vec<Session> {
        match rt().block_on(async {
            Entity::find()
                .filter(Column::TodoId.eq(todo_id))
                .order_by_asc(Column::Id)
                .all(db)
                .await
                .map_err(io_err)
        }) {
            Ok(models) => models.into_iter().map(Session::from).collect(),
            Err(e) => {
                log_error!("failed to list sessions for todo {todo_id}: {e}");
                vec![]
            }
        }
    }

    fn to_model(&self) -> ActiveModel {
        let now = now();

        match self.id {
            Some(id) => ActiveModel {
                id: Set(id),
                phase: Set(self.phase.to_db_str().to_string()),
                duration_secs: Set(self.duration_secs as i32),
                started_at: Set(format_datetime(self.started_at)),
                ended_at: Set(format_datetime(self.ended_at)),
                todo_id: Set(self.todo_id),
                created_at: Set(format_datetime(self.created_at)),
                updated_at: Set(format_datetime(now)),
            },
            None => ActiveModel {
                phase: Set(self.phase.to_db_str().to_string()),
                duration_secs: Set(self.duration_secs as i32),
                started_at: Set(format_datetime(self.started_at)),
                ended_at: Set(format_datetime(self.ended_at)),
                todo_id: Set(self.todo_id),
                created_at: Set(format_datetime(self.created_at)),
                updated_at: Set(format_datetime(now)),
                ..Default::default()
            },
        }
    }
}

impl From<Model> for Session {
    fn from(m: Model) -> Self {
        let now = now();

        Self {
            id: Some(m.id),
            phase: Phase::from_db_str(&m.phase).unwrap_or(Phase::Work),
            duration_secs: m.duration_secs.max(0) as u32,
            started_at: parse_datetime(&m.started_at).unwrap_or(now),
            ended_at: parse_datetime(&m.ended_at).unwrap_or(now),
            todo_id: m.todo_id,
            created_at: parse_datetime(&m.created_at).unwrap_or(now),
            updated_at: parse_datetime(&m.updated_at).unwrap_or(now),
        }
    }
}

#[derive(Clone, Debug, PartialEq, DeriveEntityModel)]
#[sea_orm(table_name = "sessions")]
pub struct Model {
    #[sea_orm(primary_key, auto_increment = true)]
    id: i32,
    phase: String,
    duration_secs: i32,
    started_at: String,
    ended_at: String,
    todo_id: Option<i32>,
    created_at: String,
    updated_at: String,
}

#[derive(Copy, Clone, Debug, EnumIter, DeriveRelation)]
pub enum Relation {}

impl ActiveModelBehavior for ActiveModel {}

fn io_err(e: impl std::fmt::Display) -> io::Error {
    io::Error::new(io::ErrorKind::Other, e.to_string())
}
