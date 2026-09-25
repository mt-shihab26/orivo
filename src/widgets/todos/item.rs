use ratatui::{
    prelude::{Buffer, Color, Modifier, Rect, Style, Widget},
    widgets::Paragraph,
};

use crate::{
    kinds::repeat::Repeat,
    models::{session::Stat, todo::Todo},
};

pub struct ItemProps<'a> {
    todo: &'a Todo,
    stats: Option<Stat>,
    /// 1-based serial number shown before the todo text.
    serial: usize,
    /// Width (in digits) of the widest serial number on the page, for alignment.
    serial_width: usize,
    /// Whether the row should appear dimmed (e.g. history page).
    dimmed: bool,
    selected: bool,
    /// Highlight color used when the row is selected.
    color: Color,
}

impl<'a> ItemProps<'a> {
    pub fn new(
        todo: &'a Todo,
        stats: Option<Stat>,
        serial: usize,
        serial_width: usize,
        dimmed: bool,
        selected: bool,
        color: Color,
    ) -> Self {
        Self {
            todo,
            stats,
            serial,
            serial_width,
            dimmed,
            selected,
            color,
        }
    }
}

pub struct ItemWidget<'a> {
    props: &'a ItemProps<'a>,
}

impl<'a> ItemWidget<'a> {
    pub fn new(props: &'a ItemProps<'a>) -> Self {
        Self { props }
    }

    /// Builds the display label including checkbox, repeat icon, text, stats, and due date.
    fn label(&self) -> String {
        let check = if self.props.todo.done_at.is_some() {
            "[✓]"
        } else {
            "[ ]"
        };
        let repeat_icon = if self.props.todo.repeat.is_some() {
            format!("{} ", Repeat::icon())
        } else {
            String::new()
        };

        let mut label = format!("{} {}{}", check, repeat_icon, self.props.todo.text);

        if let Some(ref stat) = self.props.stats {
            if stat.completed_sessions > 0 {
                label.push_str(&format!(
                    "  · {}× {}m",
                    stat.completed_sessions,
                    stat.completed_secs / 60
                ));
            }
        }

        if let Some(date) = self.props.todo.due_date {
            label.push_str(&format!("  [{}]", date.date()));
        }
        label
    }

    /// Returns the base text style, applying dim/strikethrough for done or history rows.
    fn base_style(&self) -> Style {
        if self.props.dimmed || self.props.todo.done_at.is_some() {
            Style::default()
                .fg(Color::DarkGray)
                .add_modifier(Modifier::CROSSED_OUT)
        } else {
            Style::default().fg(Color::White)
        }
    }
}

impl Widget for &ItemWidget<'_> {
    fn render(self, area: Rect, buf: &mut Buffer) {
        let serial = self.props.serial;
        let width = self.props.serial_width;
        let prefix = if self.props.selected { "> " } else { "  " };
        let text = format!("{prefix}{serial:>width$}. {}", self.label());
        let style = if self.props.selected {
            self.base_style()
                .fg(self.props.color)
                .add_modifier(Modifier::BOLD)
        } else {
            self.base_style()
        };
        Paragraph::new(text).style(style).render(area, buf);
    }
}
