use std::cell::{Ref, RefCell};

use sea_orm::DatabaseConnection;

use crate::{kinds::page::Page, models::todo::Todo};

/// Per-page cache for the paginated todo list and its total count.
pub struct TodosCache {
    db: DatabaseConnection,
    /// Cached page of todos, `None` until first fetch.
    items: RefCell<Option<Vec<Todo>>>,
    /// Cached total count, `None` until first fetch.
    count: RefCell<Option<usize>>,
}

impl TodosCache {
    pub fn new(db: DatabaseConnection) -> Self {
        Self {
            db,
            items: RefCell::new(None),
            count: RefCell::new(None),
        }
    }

    /// Returns the cached page of todos, querying the DB if needed.
    pub fn get_items(&self, page: Page, offset: usize, page_size: usize) -> Ref<'_, [Todo]> {
        if self.items.borrow().is_none() {
            *self.items.borrow_mut() = Some(Todo::list(&self.db, page, offset, page_size));
        }
        Ref::map(self.items.borrow(), |c| c.as_deref().unwrap_or(&[]))
    }

    /// Returns the cached todo at `idx`, querying the DB if needed.
    pub fn get_item_at(
        &self,
        page: Page,
        offset: usize,
        page_size: usize,
        idx: usize,
    ) -> Option<Ref<'_, Todo>> {
        let items = self.get_items(page, offset, page_size);
        if items.get(idx).is_none() {
            return None;
        }
        Some(Ref::map(items, |items| &items[idx]))
    }

    /// Returns the cached total count, querying the DB if needed.
    pub fn get_count(&self, page: Page) -> usize {
        let mut count = self.count.borrow_mut();
        if count.is_none() {
            *count = Some(Todo::count(&self.db, page));
        }
        count.unwrap_or(0)
    }

    /// Drops the cached todo list so the next call to `get_items()` re-queries.
    pub fn invalidate_items(&self) {
        *self.items.borrow_mut() = None;
    }

    /// Drops the cached count so the next call to `get_count()` re-queries.
    pub fn invalidate_count(&self) {
        *self.count.borrow_mut() = None;
    }

    pub fn invalidate_all(&self) {
        self.invalidate_items();
        self.invalidate_count();
    }
}
