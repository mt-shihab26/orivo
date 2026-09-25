use ratatui::{
    crossterm::event::{KeyCode, KeyEvent},
    prelude::{Buffer, Color, Modifier, Rect, Style, Stylize, Widget},
    widgets::{Block, Clear, Paragraph},
};

use crate::models::{session::Stat, todo::Todo};

pub enum TodoPickerAction {
    /// User confirmed a selection; carries the chosen todo's id.
    Select(i32),
    Cancel,
    None,
}

pub struct TodoPickerProps {
    /// Overdue todos (due_date < today).
    due_todos: Vec<Todo>,
    /// Session stats parallel to `due_todos`.
    due_stats: Vec<Stat>,
    todos: Vec<Todo>,
    /// Session stats parallel to `todos`.
    stats: Vec<Stat>,
    /// Index of the currently highlighted todo (across both sections combined).
    cursor: usize,
    /// The id of the todo currently assigned to the timer, shown in phase color.
    selected_id: Option<i32>,
    color: Color,
}

impl TodoPickerProps {
    pub fn new(
        due_todos: Vec<Todo>,
        due_stats: Vec<Stat>,
        todos: Vec<Todo>,
        stats: Vec<Stat>,
        color: Color,
        selected_id: Option<i32>,
    ) -> Self {
        // Position cursor on the currently selected todo, default to 0.
        let cursor = selected_id
            .and_then(|id| {
                due_todos.iter().position(|t| t.id == Some(id)).or_else(|| {
                    todos
                        .iter()
                        .position(|t| t.id == Some(id))
                        .map(|i| due_todos.len() + i)
                })
            })
            .unwrap_or(0);

        Self {
            due_todos,
            due_stats,
            todos,
            stats,
            cursor,
            selected_id,
            color,
        }
    }

    fn total(&self) -> usize {
        self.due_todos.len() + self.todos.len()
    }
}

pub struct TodoPickerState {
    props: TodoPickerProps,
}

impl TodoPickerState {
    pub fn new(props: TodoPickerProps) -> Self {
        Self { props }
    }

    pub fn props(&self) -> &TodoPickerProps {
        &self.props
    }

    pub fn handle(&mut self, key: KeyEvent) -> TodoPickerAction {
        let total = self.props.total();
        match key.code {
            KeyCode::Char('j') | KeyCode::Down => {
                if total > 0 {
                    self.props.cursor = (self.props.cursor + 1).min(total - 1);
                }
                TodoPickerAction::None
            }
            KeyCode::Char('k') | KeyCode::Up => {
                self.props.cursor = self.props.cursor.saturating_sub(1);
                TodoPickerAction::None
            }
            KeyCode::Enter => {
                let due_len = self.props.due_todos.len();
                let cursor = self.props.cursor;
                let id = if cursor < due_len {
                    self.props.due_todos.get(cursor).and_then(|t| t.id)
                } else {
                    self.props.todos.get(cursor - due_len).and_then(|t| t.id)
                };
                if let Some(id) = id {
                    TodoPickerAction::Select(id)
                } else {
                    TodoPickerAction::Cancel
                }
            }
            KeyCode::Esc => TodoPickerAction::Cancel,
            _ => TodoPickerAction::None,
        }
    }
}

pub struct TodoPickerWidget<'a> {
    props: &'a TodoPickerProps,
}

impl<'a> TodoPickerWidget<'a> {
    pub fn new(props: &'a TodoPickerProps) -> Self {
        Self { props }
    }
}

/// A display row: either a non-selectable section header or a selectable todo item.
enum Row {
    Header(&'static str),
    /// Logical index into the combined due + today list.
    Item(usize),
}

impl Widget for &TodoPickerWidget<'_> {
    fn render(self, area: Rect, buf: &mut Buffer) {
        let popup = centered_rect(area, 60, area.height.saturating_sub(4));

        Clear.render(popup, buf);

        let block = Block::bordered()
            .title(" Select Todo ")
            .title_bottom(" [j/k] ↑↓  [Enter] confirm  [Esc] cancel ")
            .border_style(Style::default().fg(self.props.color).bold());

        let inner = block.inner(popup);
        block.render(popup, buf);

        let total = self.props.total();

        if total == 0 {
            Paragraph::new("No todos for today")
                .centered()
                .fg(Color::DarkGray)
                .render(inner, buf);
            return;
        }

        // Build display rows (headers + items)
        let mut rows: Vec<Row> = vec![];
        if !self.props.due_todos.is_empty() {
            rows.push(Row::Header("Overdue"));
            for i in 0..self.props.due_todos.len() {
                rows.push(Row::Item(i));
            }
        }
        if !self.props.todos.is_empty() {
            rows.push(Row::Header("Today"));
            let due_len = self.props.due_todos.len();
            for i in 0..self.props.todos.len() {
                rows.push(Row::Item(due_len + i));
            }
        }

        // Find which display row the cursor item is on
        let cursor_row = rows
            .iter()
            .position(|r| matches!(r, Row::Item(i) if *i == self.props.cursor))
            .unwrap_or(0);

        let visible = inner.height as usize;
        let serial_width = total.max(1).to_string().len();
        let due_len = self.props.due_todos.len();

        // Center scroll around the cursor's display row
        let start = cursor_row
            .saturating_sub(visible / 2)
            .min(rows.len().saturating_sub(visible));

        for (row_offset, row) in rows.iter().skip(start).take(visible).enumerate() {
            let row_rect = Rect {
                y: inner.y + row_offset as u16,
                height: 1,
                ..inner
            };
            match row {
                Row::Header(label) => {
                    Paragraph::new(format!("── {label} ──"))
                        .fg(Color::DarkGray)
                        .render(row_rect, buf);
                }
                Row::Item(logical_idx) => {
                    let (todo, stat) = if *logical_idx < due_len {
                        (
                            &self.props.due_todos[*logical_idx],
                            &self.props.due_stats[*logical_idx],
                        )
                    } else {
                        let i = logical_idx - due_len;
                        (&self.props.todos[i], &self.props.stats[i])
                    };
                    let serial = logical_idx + 1;
                    let label = if stat.completed_sessions > 0 {
                        format!(
                            "{}  ·  {}× {}m",
                            todo.text,
                            stat.completed_sessions,
                            stat.completed_secs / 60
                        )
                    } else {
                        todo.text.clone()
                    };

                    let is_cursor = *logical_idx == self.props.cursor;
                    let is_selected = todo.id.is_some() && todo.id == self.props.selected_id;

                    let prefix = if is_cursor { ">" } else { " " };
                    let text = format!("{prefix} {serial:>serial_width$}. {label}");

                    if is_selected {
                        // Fill row with phase color background, render text in black
                        buf.set_style(
                            row_rect,
                            Style::default()
                                .bg(self.props.color)
                                .fg(Color::Black)
                                .add_modifier(Modifier::BOLD),
                        );
                        Paragraph::new(text).render(row_rect, buf);
                    } else if is_cursor {
                        buf.set_style(
                            row_rect,
                            Style::default()
                                .fg(self.props.color)
                                .add_modifier(Modifier::BOLD),
                        );
                        Paragraph::new(text).render(row_rect, buf);
                    } else {
                        Paragraph::new(text).fg(Color::White).render(row_rect, buf);
                    }
                }
            }
        }
    }
}

/// Computes a centered popup rect of the given dimensions within `area`.
fn centered_rect(area: Rect, width: u16, height: u16) -> Rect {
    Rect {
        x: area.x + area.width.saturating_sub(width) / 2,
        y: area.y + area.height.saturating_sub(height) / 2,
        width: width.min(area.width),
        height: height.min(area.height),
    }
}
