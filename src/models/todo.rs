use std::io;

use sea_orm::{
    ActiveModelBehavior, ActiveModelTrait, ActiveValue::Set, ColumnTrait, Condition, DatabaseConnection,
    DeriveEntityModel, DerivePrimaryKey, DeriveRelation, EntityTrait, EnumIter, Order, PaginatorTrait, PrimaryKeyTrait,
    QueryFilter, QueryOrder, QuerySelect, sea_query::Expr,
};
use time::{Duration, OffsetDateTime};

use crate::{
    kinds::{page::Page, repeat::Repeat},
    log_error, log_warn,
    utils::{
        date::{format_date, format_datetime, now, parse_datetime, today},
        db::rt,
    },
};

/// Represents a single todo item with optional scheduling and repeat configuration.
#[derive(Clone)]
pub struct Todo {
    /// Database primary key; `None` until the todo is persisted.
    pub id: Option<i32>,
    /// The todo's display text.
    pub text: String,
    /// Local datetime when the todo was completed, or `None` if not yet done.
    pub done_at: Option<OffsetDateTime>,
    /// Optional datetime the todo is due.
    pub due_date: Option<OffsetDateTime>,
    /// Optional repeat schedule applied when the todo is completed.
    pub repeat: Option<Repeat>,
    /// Id of the todo this was spawned from, if it is a repeated occurrence.
    pub parent_id: Option<i32>,
    /// Local datetime when the todo was created.
    pub created_at: OffsetDateTime,
    /// Local datetime when the todo was last updated.
    pub updated_at: OffsetDateTime,
}

impl Todo {
    /// Creates an unsaved in-memory Todo with default values.
    pub fn new(text: String, due_date: Option<OffsetDateTime>, repeat: Option<Repeat>, parent_id: Option<i32>) -> Self {
        let now = now();

        Self {
            id: None,
            text,
            done_at: None,
            due_date,
            repeat,
            parent_id,
            created_at: now,
            updated_at: now,
        }
    }

    /// Inserts or updates the todo in the database, returning whether it succeeded.
    pub fn save(&mut self, db: &DatabaseConnection) -> bool {
        match self.id {
            Some(_) => self.update(db),
            None => match rt().block_on(async { self.to_model().insert(db).await.map_err(io_err) }) {
                Ok(model) => {
                    *self = model.into();
                    true
                }
                Err(e) => {
                    log_error!("failed to insert todo: {e}");
                    false
                }
            },
        }
    }

    /// Creates the next repeated occurrence of this todo, skipping if one already exists.
    pub fn save_next(&self, db: &DatabaseConnection) -> Option<Todo> {
        let (Some(repeat), Some(due_date)) = (self.repeat.as_ref(), self.due_date) else {
            return None;
        };

        let mut next_date = repeat.next_date(due_date.date());
        let today_date = today();
        while next_date <= today_date {
            next_date = repeat.next_date(next_date);
        }
        let next_date_str = format_date(next_date);
        let parent_id = self.id;

        let already_exists = rt().block_on(async {
            Entity::find()
                .filter(Column::ParentId.eq(parent_id))
                .filter(Column::DueDate.eq(&next_date_str))
                .count(db)
                .await
                .unwrap_or(0)
        }) > 0;

        if already_exists {
            return None;
        }

        let next_dt = due_date.replace_date(next_date);
        let mut next = Todo::new(self.text.clone(), Some(next_dt), Some(Repeat::of(repeat)), parent_id);

        if next.save(db) { Some(next) } else { None }
    }

    /// Returns today's todos that are not yet done, for the timer picker.
    pub fn list_today_pending(db: &DatabaseConnection) -> Vec<Todo> {
        let today = format_date(today());
        let query = Entity::find()
            .filter(Expr::cust_with_values("substr(due_date, 1, 10) = ?", [today]))
            .filter(Column::DoneAt.is_null())
            .order_by_desc(Column::CreatedAt);

        match rt().block_on(async { query.all(db).await.map_err(io_err) }) {
            Ok(models) => models.into_iter().map(Todo::from).collect(),
            Err(e) => {
                log_error!("failed to load today's pending todos: {e}");
                vec![]
            }
        }
    }

    /// Returns overdue todos that are not yet done (due_date < today), for the timer picker.
    pub fn list_due_pending(db: &DatabaseConnection) -> Vec<Todo> {
        let today = format_date(today());
        let query = Entity::find()
            .filter(Expr::cust_with_values("substr(due_date, 1, 10) < ?", [today]))
            .filter(Column::DoneAt.is_null())
            .order_by_desc(Column::DueDate)
            .order_by_desc(Column::CreatedAt);

        match rt().block_on(async { query.all(db).await.map_err(io_err) }) {
            Ok(models) => models.into_iter().map(Todo::from).collect(),
            Err(e) => {
                log_error!("failed to load overdue pending todos: {e}");
                vec![]
            }
        }
    }

    fn base_search_query(page: Page, query: &str) -> sea_orm::Select<Entity> {
        let pattern = format!("%{}%", query.to_lowercase());
        Self::base_query(page).filter(Expr::cust_with_values("lower(text) LIKE ?", [pattern]))
    }

    /// Returns a paginated list of todos matching `query` (case-insensitive) on the given page.
    pub fn list_search(db: &DatabaseConnection, page: Page, query: &str, offset: usize, limit: usize) -> Vec<Todo> {
        if limit == 0 {
            return vec![];
        }
        let query_stmt = Self::base_search_query(page, query)
            .offset(offset as u64)
            .limit(limit as u64);
        match rt().block_on(async { query_stmt.all(db).await.map_err(io_err) }) {
            Ok(models) => models.into_iter().map(Todo::from).collect(),
            Err(e) => {
                log_error!("failed to search todos: {e}");
                vec![]
            }
        }
    }

    /// Returns the total number of todos matching `query` (case-insensitive) on the given page.
    pub fn count_search(db: &DatabaseConnection, page: Page, query: &str) -> usize {
        let query_stmt = Self::base_search_query(page, query);
        match rt().block_on(async { query_stmt.count(db).await.map_err(io_err) }) {
            Ok(count) => count as usize,
            Err(e) => {
                log_error!("failed to count search results: {e}");
                0
            }
        }
    }

    /// Returns a paginated list of todos for the given page filter.
    pub fn list(db: &DatabaseConnection, page: Page, offset: usize, limit: usize) -> Vec<Todo> {
        if limit == 0 {
            return vec![];
        }

        let query = Self::base_query(page).offset(offset as u64).limit(limit as u64);

        match rt().block_on(async { query.all(db).await.map_err(io_err) }) {
            Ok(models) => models.into_iter().map(Todo::from).collect(),
            Err(e) => {
                log_error!(
                    "failed to load todos for page {} (offset={}, limit={}): {e}",
                    page.label(),
                    offset,
                    limit
                );
                vec![]
            }
        }
    }

    /// Returns the total number of todos for the given page filter.
    pub fn count(db: &DatabaseConnection, page: Page) -> usize {
        let query = Self::base_query(page);

        match rt().block_on(async { query.count(db).await.map_err(io_err) }) {
            Ok(count) => count as usize,
            Err(e) => {
                log_error!("failed to count todos for page {}: {e}", page.label());
                0
            }
        }
    }

    /// Counts items that appear before today in the timeline sort order (past dates + no-date).
    /// `done` filters by completion state: `false` for Index, `true` for History.
    pub fn count_before_today(db: &DatabaseConnection, done: bool) -> usize {
        let today = format_date(today());
        let done_filter = if done {
            Column::DoneAt.is_not_null()
        } else {
            Column::DoneAt.is_null()
        };
        let query = Entity::find().filter(done_filter).filter(
            Condition::any()
                .add(Expr::cust_with_values("substr(due_date, 1, 10) < ?", [&today]))
                .add(Column::DueDate.is_null()),
        );
        match rt().block_on(async { query.count(db).await.map_err(io_err) }) {
            Ok(count) => count as usize,
            Err(e) => {
                log_error!("failed to count timeline items before today: {e}");
                0
            }
        }
    }

    /// Builds the base ORM query for the given page filter.
    fn base_query(page: Page) -> sea_orm::Select<Entity> {
        let today_date = today();
        let today = format_date(today_date);

        match page {
            Page::Due => Entity::find()
                .filter(Column::DoneAt.is_null())
                .filter(Expr::cust_with_values("substr(due_date, 1, 10) < ?", [today.clone()]))
                .order_by_desc(Column::DueDate)
                .order_by_desc(Column::CreatedAt),
            Page::Today => Entity::find()
                .filter(Expr::cust_with_values("substr(due_date, 1, 10) = ?", [today.clone()]))
                .order_by_desc(Column::CreatedAt),
            Page::Index => {
                let null_key = format!("{}~", format_date(today_date - Duration::days(1)));
                Entity::find()
                    .filter(Column::DoneAt.is_null())
                    .order_by(
                        Expr::cust_with_values("CASE WHEN due_date IS NULL THEN ? ELSE due_date END", [null_key]),
                        Order::Asc,
                    )
                    .order_by_asc(Column::CreatedAt)
            }
            Page::History => {
                let null_key = format!("{}~", format_date(today_date - Duration::days(1)));
                Entity::find()
                    .filter(Column::DoneAt.is_not_null())
                    .order_by(
                        Expr::cust_with_values("CASE WHEN due_date IS NULL THEN ? ELSE due_date END", [null_key]),
                        Order::Asc,
                    )
                    .order_by_asc(Column::CreatedAt)
            }
        }
    }

    /// Persists the current field values to the database, returning whether it succeeded.
    pub fn update(&mut self, db: &DatabaseConnection) -> bool {
        match rt().block_on(async { self.to_model().update(db).await.map_err(io_err) }) {
            Ok(model) => {
                *self = model.into();
                true
            }
            Err(e) => {
                log_error!("failed to update todo: {e}");
                false
            }
        }
    }

    /// Toggles the done state and spawns the next repeat occurrence when marked done.
    pub fn toggle(&mut self, db: &DatabaseConnection) -> Option<Todo> {
        let prev_done_at = self.done_at;

        if self.done_at.is_some() {
            self.done_at = None;
        } else {
            self.done_at = Some(now());
            if self.due_date.is_none() {
                self.due_date = Some(now());
            }
        }

        if !self.update(db) {
            self.done_at = prev_done_at;
            return None;
        }

        if self.done_at.is_none() {
            return None;
        }

        self.save_next(db)
    }

    /// Deletes this todo from the database, returning whether it succeeded.
    pub fn delete(&self, db: &DatabaseConnection) -> bool {
        let Some(id) = self.id else {
            log_warn!("todo has no id, skipping db delete");
            return false;
        };

        match rt().block_on(async { Entity::delete_by_id(id).exec(db).await.map_err(io_err).map(|_| ()) }) {
            Ok(()) => true,
            Err(e) => {
                log_error!("failed to delete todo: {e}");
                false
            }
        }
    }

    /// Converts this todo into a SeaORM active model for insert or update.
    fn to_model(&self) -> ActiveModel {
        let due_date = self.due_date.map(|dt| format_date(dt.date()));
        let repeat = self.repeat.as_ref().map(|r| r.to_db_str().to_string());
        let now = format_datetime(now());
        match self.id {
            Some(id) => ActiveModel {
                id: Set(id),
                text: Set(self.text.clone()),
                done_at: Set(self.done_at.map(format_datetime)),
                due_date: Set(due_date),
                repeat: Set(repeat),
                parent_id: Set(self.parent_id),
                created_at: Set(format_datetime(self.created_at)),
                updated_at: Set(now),
            },
            None => ActiveModel {
                text: Set(self.text.clone()),
                done_at: Set(self.done_at.map(format_datetime)),
                due_date: Set(due_date),
                repeat: Set(repeat),
                parent_id: Set(self.parent_id),
                created_at: Set(now.clone()),
                updated_at: Set(now),
                ..Default::default()
            },
        }
    }
}

impl From<Model> for Todo {
    /// Converts a SeaORM todo row into the domain `Todo` type.
    fn from(m: Model) -> Self {
        Self {
            id: Some(m.id),
            text: m.text,
            done_at: m.done_at.as_deref().and_then(parse_datetime),
            due_date: m.due_date.as_deref().and_then(parse_datetime),
            repeat: m.repeat.as_deref().and_then(Repeat::from_db_str),
            parent_id: m.parent_id,
            created_at: parse_datetime(&m.created_at).unwrap_or_else(now),
            updated_at: parse_datetime(&m.updated_at).unwrap_or_else(now),
        }
    }
}

/// SeaORM row model for the `todos` table.
#[derive(Clone, Debug, PartialEq, DeriveEntityModel)]
#[sea_orm(table_name = "todos")]
pub struct Model {
    #[sea_orm(primary_key, auto_increment = true)]
    id: i32,
    text: String,
    done_at: Option<String>,
    due_date: Option<String>,
    repeat: Option<String>,
    parent_id: Option<i32>,
    created_at: String,
    updated_at: String,
}

/// SeaORM relation set for `todos`.
#[derive(Copy, Clone, Debug, EnumIter, DeriveRelation)]
pub enum Relation {}

impl ActiveModelBehavior for ActiveModel {}

/// Normalizes database and ORM errors into `io::Error` for shared logging paths.
fn io_err(e: impl std::fmt::Display) -> io::Error {
    io::Error::new(io::ErrorKind::Other, e.to_string())
}
