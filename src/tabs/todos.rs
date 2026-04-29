use std::{
    cell::Ref,
    io::Result,
    sync::{Arc, Mutex},
};

use ratatui::{
    Frame,
    crossterm::event::{KeyCode, KeyEvent, KeyModifiers},
    prelude::{Color, Constraint, Layout, Rect, Widget},
};
use sea_orm::DatabaseConnection;

use crate::{
    caches::timer::TimerCache,
    kinds::{page::Page, todos_mode::TodosMode},
    models::todo::Todo,
    states::todos::TodosState,
    utils::date::now,
    widgets::{
        layout::border::{BorderProps, BorderWidget},
        todos::{
            hint::{HintProps, HintWidget},
            input::{InputAction, InputProps, InputState, InputWidget},
            list::{ListProps, ListWidget},
            search::{SearchAction, SearchProps, SearchState, SearchWidget},
            status::{StatusProps, StatusWidget},
            tabs::{TabsProps, TabsWidget},
        },
    },
};

use super::Tab;

/// Accent color for the todos tab UI elements.
pub const COLOR: Color = Color::Green;

/// The todos tab, managing the task list UI and input state.
pub struct TodosTab {
    /// Currently active page view (Due, Today, Index, History).
    page: Page,
    /// Current UI mode controlling input handling.
    mode: TodosMode,
    /// Underlying todos state holding data and pagination.
    state: TodosState,
    /// Active text input state when adding or editing a todo.
    input_state: Option<InputState>,
    /// Active search bar state when in Searching mode.
    search_state: Option<SearchState>,
}

impl TodosTab {
    /// Creates a new `TodosTab` connected to the given database and timer cache.
    pub fn new(db: DatabaseConnection, timer_cache: Arc<Mutex<TimerCache>>) -> Self {
        Self {
            page: Page::Today,
            mode: TodosMode::Normal,
            state: TodosState::new(db, timer_cache),
            input_state: None,
            search_state: None,
        }
    }

    /// Returns the visible todo items for the current page.
    fn items(&self) -> Ref<'_, [Todo]> {
        self.state.items(self.page)
    }

    /// Returns the total number of todos on the current page.
    fn count(&self) -> usize {
        self.state.count(self.page)
    }

    /// Switches to the given page and resets its pagination state.
    fn set_page(&mut self, page: Page) {
        self.page = page;
        self.state.reset_page(self.page);
    }

    /// Discards the active input and returns to normal mode.
    fn cancel_input(&mut self) {
        self.input_state = None;
        self.mode = TodosMode::Normal;
    }

    /// Opens the search bar pre-filled with any active query.
    fn open_search(&mut self) {
        self.search_state = Some(SearchState::new(self.state.search_query()));
        self.mode = TodosMode::Searching;
    }

    /// Closes the search bar, keeping the current filter active.
    fn confirm_search(&mut self) {
        self.search_state = None;
        self.mode = TodosMode::Normal;
    }

    /// Closes the search bar and clears the filter entirely.
    fn cancel_search(&mut self) {
        self.state.clear_search();
        self.search_state = None;
        self.mode = TodosMode::Normal;
    }
}

impl Tab for TodosTab {
    /// Returns the tab label shown in the tab bar.
    fn name(&self) -> &str {
        "Todos [^t]"
    }

    /// Returns the accent color for the todos tab.
    fn color(&self) -> Color {
        COLOR
    }

    /// Handles a key event, delegating to input, search, or normal-mode handlers.
    fn handle(&mut self, key: KeyEvent) -> Result<()> {
        let pending_g = self.state.begin_input();

        match self.mode {
            TodosMode::Normal => match key.code {
                KeyCode::Char('1') => self.set_page(Page::Due),
                KeyCode::Char('2') => self.set_page(Page::Today),
                KeyCode::Char('3') => self.set_page(Page::Index),
                KeyCode::Char('4') => self.set_page(Page::History),
                KeyCode::Char(']') => self.set_page(self.page.next()),
                KeyCode::Char('[') => self.set_page(self.page.prev()),
                KeyCode::Char('j') | KeyCode::Down => self.state.move_selection(self.page, 1),
                KeyCode::Char('k') | KeyCode::Up => self.state.move_selection(self.page, -1),
                KeyCode::Char('g') => self.state.go_to_start(pending_g),
                KeyCode::Char('G') => self.state.go_to_end(self.page),
                KeyCode::Char(' ') | KeyCode::Enter => self.state.toggle_selected(self.page),
                KeyCode::Char('x') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                    self.state.delete_selected(self.page)
                }
                KeyCode::Char('a') => {
                    self.mode = TodosMode::Adding;
                    let date = if self.page == Page::Today { Some(now()) } else { None };
                    self.input_state = Some(InputState::new(InputProps::new(None, date, None)));
                }
                KeyCode::Char('e') => {
                    if let Some((text, due_date, repeat)) = self.state.edit_values(self.page) {
                        self.mode = TodosMode::Editing;
                        self.input_state =
                            Some(InputState::new(InputProps::new(Some(&text), due_date, repeat.as_ref())));
                    }
                }
                KeyCode::Char('/') => self.open_search(),
                KeyCode::Esc => {
                    if self.state.is_searching() {
                        self.state.clear_search();
                    }
                }
                _ => {}
            },
            TodosMode::Adding => {
                if let Some(input_widget) = &mut self.input_state {
                    match input_widget.handle(key) {
                        InputAction::Confirm { text, date, repeat } => {
                            self.state.add(self.page, text, date, repeat);
                            self.cancel_input();
                        }
                        InputAction::Escape => self.cancel_input(),
                        InputAction::None => {}
                    }
                }
            }
            TodosMode::Editing => {
                if let Some(input_widget) = &mut self.input_state {
                    match input_widget.handle(key) {
                        InputAction::Confirm { text, date, repeat } => {
                            self.state.update(self.page, text, date, repeat);
                            self.cancel_input();
                        }
                        InputAction::Escape => self.cancel_input(),
                        InputAction::None => {}
                    }
                }
            }
            TodosMode::Searching => {
                if let Some(search) = &mut self.search_state {
                    match search.handle(key) {
                        SearchAction::Confirm => self.confirm_search(),
                        SearchAction::Cancel => self.cancel_search(),
                        SearchAction::QueryChanged(query) => {
                            self.state.set_search_query(query, self.page);
                        }
                        SearchAction::None => {}
                    }
                }
            }
        }
        self.state.sync_list_state(self.items().len());
        Ok(())
    }

    /// Renders the todos tab including the list, tabs bar, hint, and input/search overlay.
    fn render(&self, frame: &mut Frame, area: Rect) {
        let buf = frame.buffer_mut();

        let area = BorderWidget::new(&BorderProps::new(self.color()), area).render(area, buf);

        let (tabs_area, list_area, hint_area, bottom_area) =
            if matches!(self.mode, TodosMode::Adding | TodosMode::Editing) {
                let [tabs, list, hint, bottom] = Layout::vertical([
                    Constraint::Length(1),
                    Constraint::Fill(1),
                    Constraint::Length(1),
                    Constraint::Length(3),
                ])
                .areas(area);
                (tabs, list, hint, Some(bottom))
            } else if matches!(self.mode, TodosMode::Searching) || self.state.is_searching() {
                let [tabs, list, hint, bottom] = Layout::vertical([
                    Constraint::Length(1),
                    Constraint::Fill(1),
                    Constraint::Length(1),
                    Constraint::Length(1),
                ])
                .areas(area);
                (tabs, list, hint, Some(bottom))
            } else {
                let [tabs, list, hint] =
                    Layout::vertical([Constraint::Length(1), Constraint::Fill(1), Constraint::Length(1)]).areas(area);
                (tabs, list, hint, None)
            };

        self.state.set_visible_capacity(list_area);
        let items = self.items();
        let stats = self.state.stats(self.page);
        let total = self.count();
        let from = self.state.from(total);
        let to = self.state.to(items.len());
        let page = self.state.page();

        StatusWidget::new(&StatusProps::new(total, from, to, page)).render(area, buf);
        TabsWidget::new(&TabsProps::new(self.page, self.color())).render(tabs_area, buf);
        ListWidget::new(&ListProps::new(
            &items,
            &stats,
            self.state.offset(),
            self.page,
            self.state.selected(),
            self.color(),
            self.state.show_more_above(),
            self.state.show_more_below(items.len(), total),
        ))
        .render(list_area, buf);
        HintWidget::new(&HintProps::new(
            self.mode,
            self.state.can_delete(self.page, &items),
            self.state.is_searching(),
        ))
        .render(hint_area, buf);

        if let Some(bottom_rect) = bottom_area {
            match self.mode {
                TodosMode::Adding | TodosMode::Editing => {
                    if let Some(input_state) = &self.input_state {
                        InputWidget::new(input_state.props()).render(bottom_rect, buf);
                        input_state.render_calendar(area, buf);
                        input_state.render_repeat(area, buf);
                    }
                }
                TodosMode::Searching => {
                    if let Some(search_state) = &self.search_state {
                        SearchWidget::new(search_state.props()).render(bottom_rect, buf);
                    }
                }
                TodosMode::Normal => {
                    let props = SearchProps::new(self.state.search_query(), false);
                    SearchWidget::new(&props).render(bottom_rect, buf);
                }
            }
        }
    }

    /// Drops any cached data held by this tab.
    fn invalidate_cache(&mut self) {
        self.state.refresh(self.page);
    }
}
