use std::{
    cell::{Cell, Ref, RefCell, RefMut},
    sync::{Arc, Mutex},
};

use ratatui::{layout::Rect, widgets::ListState};
use sea_orm::DatabaseConnection;
use time::OffsetDateTime;

use crate::{
    caches::{timer::TimerCache, todos::TodosCache},
    kinds::{page::Page, repeat::Repeat},
    models::{
        session::{Session, Stat},
        todo::Todo,
    },
};

/// Runtime state for the todos tab, managing pagination, selection, and data caches.
pub struct TodosState {
    db: DatabaseConnection,
    /// Whether a `g` keypress is pending (waiting for a second `g` to jump to top).
    pending_g: bool,
    /// Index of the currently selected row within the visible page.
    selected: usize,
    /// Global offset into the full todo list for the current page view.
    offset: usize,
    /// Number of items that fit in the visible list area; updated on each resize.
    page_size: Cell<usize>,
    list_state: RefCell<ListState>,
    /// Cache for the todos list, keyed by page and pagination params.
    todos_cache: TodosCache,
    /// Shared timer cache, invalidated when todos change.
    timer_cache: Arc<Mutex<TimerCache>>,
    /// Active search query; empty string means no search is active.
    search_query: String,
    /// Cached search result page; invalidated when query or offset changes.
    search_items: RefCell<Option<Vec<Todo>>>,
    /// Cached total count of search results; invalidated when query or data changes.
    search_count: RefCell<Option<usize>>,
}

impl TodosState {
    pub fn new(db: DatabaseConnection, timer_cache: Arc<Mutex<TimerCache>>) -> Self {
        Self {
            todos_cache: TodosCache::new(db.clone()),
            db,
            pending_g: false,
            selected: 0,
            offset: 0,
            page_size: Cell::new(1),
            list_state: RefCell::new(ListState::default()),
            timer_cache,
            search_query: String::new(),
            search_items: RefCell::new(None),
            search_count: RefCell::new(None),
        }
    }

    pub fn selected(&self) -> usize {
        self.selected
    }

    pub fn offset(&self) -> usize {
        self.offset
    }

    pub fn list_state_mut(&self) -> RefMut<'_, ListState> {
        self.list_state.borrow_mut()
    }

    /// Consumes and returns the pending-`g` flag, resetting it to `false`.
    pub fn begin_input(&mut self) -> bool {
        let pending_g = self.pending_g;
        self.pending_g = false;
        pending_g
    }

    /// Returns the 1-based index of the first visible item, or 0 if the list is empty.
    pub fn from(&self, total: usize) -> usize {
        if total == 0 { 0 } else { self.offset + 1 }
    }

    /// Returns the 1-based index of the last visible item.
    pub fn to(&self, loaded_len: usize) -> usize {
        self.offset + loaded_len
    }

    /// Returns the current 1-based page number.
    pub fn page(&self) -> usize {
        (self.offset / self.page_size()) + 1
    }

    pub fn show_more_above(&self) -> bool {
        self.offset > 0
    }

    pub fn show_more_below(&self, loaded_len: usize, total: usize) -> bool {
        self.offset + loaded_len < total
    }

    pub fn search_query(&self) -> &str {
        &self.search_query
    }

    pub fn is_searching(&self) -> bool {
        !self.search_query.is_empty()
    }

    /// Sets the search query and resets pagination; re-fetches results on the next access.
    pub fn set_search_query(&mut self, query: String, page: Page) {
        if self.search_query == query {
            return;
        }
        self.search_query = query;
        self.invalidate_search_caches();
        self.offset = 0;
        self.selected = 0;
        self.clamp_selected(page);
    }

    pub fn clear_search(&mut self) {
        self.search_query.clear();
        self.invalidate_search_caches();
    }

    fn invalidate_search_caches(&self) {
        *self.search_items.borrow_mut() = None;
        *self.search_count.borrow_mut() = None;
    }

    fn get_search_items(&self, page: Page) -> Ref<'_, [Todo]> {
        if self.search_items.borrow().is_none() {
            *self.search_items.borrow_mut() = Some(Todo::list_search(
                &self.db,
                page,
                &self.search_query,
                self.offset,
                self.page_size(),
            ));
        }
        Ref::map(self.search_items.borrow(), |c| c.as_deref().unwrap_or(&[]))
    }

    /// Returns session stats for each visible todo on the given page.
    pub fn stats(&self, page: Page) -> Vec<Option<Stat>> {
        self.items(page)
            .iter()
            .map(|t| t.id.map(|id| Session::stat(&self.db, id)))
            .collect()
    }

    pub fn items(&self, page: Page) -> Ref<'_, [Todo]> {
        if self.search_query.is_empty() {
            self.todos_cache
                .get_items(page, self.offset, self.page_size())
        } else {
            self.get_search_items(page)
        }
    }

    pub fn count(&self, page: Page) -> usize {
        if self.search_query.is_empty() {
            self.todos_cache.get_count(page)
        } else {
            let mut count = self.search_count.borrow_mut();
            if count.is_none() {
                *count = Some(Todo::count_search(&self.db, page, &self.search_query));
            }
            count.unwrap_or(0)
        }
    }

    pub fn selected_item(&self, page: Page) -> Option<Ref<'_, Todo>> {
        if self.search_query.is_empty() {
            self.todos_cache
                .get_item_at(page, self.offset, self.page_size(), self.selected)
        } else {
            let items = self.get_search_items(page);
            if items.get(self.selected).is_none() {
                return None;
            }
            Some(Ref::map(items, |items| &items[self.selected]))
        }
    }

    /// Updates the visible capacity based on the rendered list area and invalidates caches on change.
    pub fn set_visible_capacity(&self, list_area: Rect) {
        let top_padding = 1usize;
        let capacity = list_area.height.saturating_sub(top_padding as u16) as usize;
        let capacity = capacity.max(1);

        if self.page_size.get() != capacity {
            self.page_size.set(capacity);
            self.clear_caches();
        }
    }

    /// Returns the current page size, ensuring it is at least 1.
    pub fn page_size(&self) -> usize {
        self.page_size.get().max(1)
    }

    pub fn clear_caches(&self) {
        self.todos_cache.invalidate_all();
        self.invalidate_search_caches();
        self.invalidate_timer_todos();
    }

    /// Invalidates the timer cache's todo list so the picker reflects recent changes.
    fn invalidate_timer_todos(&self) {
        if let Ok(mut c) = self.timer_cache.lock() {
            c.invalidate_todos();
        }
    }

    /// Invalidates only the items cache (keeping counts), including the search items page.
    fn invalidate_items(&self) {
        self.todos_cache.invalidate_items();
        *self.search_items.borrow_mut() = None;
    }

    pub fn can_delete(&self, page: Page, items: &[Todo]) -> bool {
        !matches!(page, Page::History)
            && items
                .get(self.selected)
                .is_some_and(|todo| todo.done_at.is_none())
    }

    /// Clamps the selection to valid bounds, scrolling back a page if the current page is empty.
    pub fn clamp_selected(&mut self, page: Page) {
        let mut len = self.items(page).len();
        if len == 0 && self.offset > 0 {
            self.offset = self.offset.saturating_sub(self.page_size());
            self.invalidate_items();
            len = self.items(page).len();
        }

        if len == 0 {
            self.selected = 0;
        } else {
            self.selected = self.selected.min(len - 1);
        }
    }

    pub fn sync_list_state(&self, len: usize) {
        let selected = if len == 0 {
            None
        } else {
            Some(self.selected.min(len - 1))
        };
        self.list_state.borrow_mut().select(selected);
    }

    /// Resets pagination and selection for the given page, jumping to today's items where applicable.
    pub fn reset_page(&mut self, page: Page) {
        self.pending_g = false;
        self.offset = 0;
        self.selected = 0;
        self.clear_caches();

        let done = match page {
            Page::Index => Some(false),
            Page::History => Some(true),
            _ => None,
        };
        if let Some(done) = done {
            let before = Todo::count_before_today(&self.db, done);
            let page_size = self.page_size();
            self.offset = (before / page_size) * page_size;
            self.selected = before - self.offset;
            self.invalidate_items();
        }

        self.clamp_selected(page);
    }

    /// Jumps to the top of the list on a double `g` press.
    pub fn go_to_start(&mut self, pending_g: bool) {
        if pending_g {
            self.offset = 0;
            self.selected = 0;
            self.invalidate_items();
        }
        self.pending_g = !pending_g;
    }

    pub fn go_to_end(&mut self, page: Page) {
        let total = Todo::count(&self.db, page);
        if total == 0 {
            return;
        }
        let page_size = self.page_size();
        self.offset = total.saturating_sub(page_size);
        self.selected = (total - 1) - self.offset;
        self.invalidate_items();
    }

    /// Refreshes caches and clamps the selection after a data mutation.
    pub fn refresh(&mut self, page: Page) {
        self.clear_caches();
        self.clamp_selected(page);
    }

    pub fn add(
        &mut self,
        page: Page,
        text: String,
        due_date: Option<OffsetDateTime>,
        repeat: Option<Repeat>,
    ) {
        let mut todo = Todo::new(text, due_date, repeat, None);
        if todo.save(&self.db) {
            self.refresh(page);
        }
    }

    pub fn update(
        &mut self,
        page: Page,
        text: String,
        due_date: Option<OffsetDateTime>,
        repeat: Option<Repeat>,
    ) {
        if let Some(mut todo) = self.selected_item(page).map(|todo| todo.clone()) {
            todo.text = text;
            todo.due_date = due_date;
            todo.repeat = repeat;
            if todo.update(&self.db) {
                self.refresh(page);
            }
        }
    }

    /// Moves the selection by `delta` rows, scrolling the page when reaching the edge.
    pub fn move_selection(&mut self, page: Page, delta: isize) {
        if delta > 0 {
            for _ in 0..delta as usize {
                let len = self.items(page).len();
                if len == 0 {
                    self.selected = 0;
                    break;
                }

                if self.selected + 1 < len {
                    self.selected += 1;
                } else if len == self.page_size() {
                    self.offset += 1;
                    self.invalidate_items();
                } else {
                    break;
                }
            }
        } else if delta < 0 {
            for _ in 0..delta.unsigned_abs() {
                let len = self.items(page).len();
                if len == 0 {
                    self.selected = 0;
                    break;
                }

                if self.selected > 0 {
                    self.selected -= 1;
                } else if self.offset > 0 {
                    self.offset -= 1;
                    self.invalidate_items();
                } else {
                    break;
                }
            }
        }
    }

    pub fn toggle_selected(&mut self, page: Page) {
        if let Some(mut todo) = self.selected_item(page).map(|todo| todo.clone()) {
            todo.toggle(&self.db);
            self.refresh(page);
        }
    }

    /// Deletes the selected todo if it is not done, then refreshes the list.
    pub fn delete_selected(&mut self, page: Page) {
        let deleted = {
            let todo = match self.selected_item(page) {
                Some(todo) => todo,
                None => return (),
            };
            if todo.done_at.is_some() {
                return ();
            }
            todo.delete(&self.db)
        };

        if deleted {
            self.refresh(page);
        }
    }

    /// Returns the text, due date, and repeat rule of the selected todo for editing.
    pub fn edit_values(
        &self,
        page: Page,
    ) -> Option<(String, Option<OffsetDateTime>, Option<Repeat>)> {
        self.selected_item(page)
            .map(|todo| (todo.text.clone(), todo.due_date, todo.repeat.clone()))
    }
}
