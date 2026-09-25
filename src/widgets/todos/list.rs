use ratatui::{
    prelude::{Buffer, Color, Line, Modifier, Rect, Span, Style, Widget},
    widgets::Paragraph,
};
use time::Duration;

use crate::{
    kinds::page::Page,
    models::{session::Stat, todo::Todo},
    utils::date::today,
};

use super::{
    indicator::{IndicatorProps, IndicatorWidget},
    item::{ItemProps, ItemWidget},
};

pub struct ListProps<'a> {
    items: &'a [Todo],
    /// Session stats aligned by index with `items`; `None` when unavailable.
    stats: &'a [Option<Stat>],
    /// Global offset of the first item in `items`, used for serial numbering.
    offset: usize,
    /// Active page, which controls the rendering layout.
    page: Page,
    /// Index within `items` of the currently selected row.
    selected: usize,
    /// Accent color for the selected row and section headers.
    color: Color,
    show_more_above: bool,
    show_more_below: bool,
}

impl<'a> ListProps<'a> {
    pub fn new(
        items: &'a [Todo],
        stats: &'a [Option<Stat>],
        offset: usize,
        page: Page,
        selected: usize,
        color: Color,
        show_more_above: bool,
        show_more_below: bool,
    ) -> Self {
        Self {
            items,
            stats,
            offset,
            page,
            selected,
            color,
            show_more_above,
            show_more_below,
        }
    }
}

pub struct ListWidget<'a> {
    props: &'a ListProps<'a>,
}

impl<'a> ListWidget<'a> {
    pub fn new(props: &'a ListProps<'a>) -> Self {
        Self { props }
    }
}

impl ListWidget<'_> {
    /// Renders todos as a plain sequential list without date section headers.
    fn render_flat(&self, area: Rect, buf: &mut Buffer) {
        let dimmed = matches!(self.props.page, Page::History);
        let serial_width = (self.props.offset + self.props.items.len())
            .max(1)
            .to_string()
            .len();

        for (index, todo) in self.props.items.iter().enumerate() {
            let y = area.y + index as u16;
            if y >= area.y + area.height {
                break;
            }
            let stats = self.props.stats.get(index).cloned().flatten();
            ItemWidget::new(&ItemProps::new(
                todo,
                stats,
                self.props.offset + index + 1,
                serial_width,
                dimmed,
                index == self.props.selected,
                self.props.color,
            ))
            .render(
                Rect {
                    y,
                    height: 1,
                    ..area
                },
                buf,
            );
        }
    }

    /// Renders todos grouped under date section headers with virtual scrolling.
    fn render_index(&self, area: Rect, buf: &mut Buffer) {
        let serial_width = (self.props.offset + self.props.items.len())
            .max(1)
            .to_string()
            .len();

        enum Row {
            Header(Line<'static>),
            Item(usize),
        }

        let mut rows: Vec<Row> = Vec::new();
        let mut selected_row = 0;
        let mut last_date = None;

        for (index, todo) in self.props.items.iter().enumerate() {
            let todo_date = todo.due_date.map(|dt| dt.date());
            if todo_date != last_date {
                rows.push(Row::Header(section_line(
                    todo_date,
                    area.width as usize,
                    self.props.color,
                )));
                last_date = todo_date;
            }
            if index == self.props.selected {
                selected_row = rows.len();
            }
            rows.push(Row::Item(index));
        }

        let visible = area.height as usize;
        let start = if rows.len() <= visible {
            0
        } else {
            selected_row
                .saturating_sub(visible.saturating_sub(1))
                .min(rows.len().saturating_sub(visible))
        };
        let end = (start + visible).min(rows.len());

        for (row_idx, row) in rows[start..end].iter().enumerate() {
            let row_rect = Rect {
                y: area.y + row_idx as u16,
                height: 1,
                ..area
            };
            match row {
                Row::Header(line) => {
                    Paragraph::new(line.clone()).render(row_rect, buf);
                }
                Row::Item(index) => {
                    let todo = &self.props.items[*index];
                    let stats = self.props.stats.get(*index).cloned().flatten();
                    ItemWidget::new(&ItemProps::new(
                        todo,
                        stats,
                        self.props.offset + index + 1,
                        serial_width,
                        false,
                        *index == self.props.selected,
                        self.props.color,
                    ))
                    .render(row_rect, buf);
                }
            }
        }
    }
}

impl Widget for &ListWidget<'_> {
    fn render(self, area: Rect, buf: &mut Buffer) {
        let horizontal_padding = 2;
        let top_padding = 1;
        let bottom_padding = 1;
        let padded_area = Rect {
            x: area.x + horizontal_padding,
            y: area.y + top_padding,
            width: area.width.saturating_sub(horizontal_padding * 2),
            height: area.height.saturating_sub(top_padding + bottom_padding),
        };

        if padded_area.height == 0 {
            return;
        }

        match self.props.page {
            Page::Index | Page::History => self.render_index(padded_area, buf),
            Page::Due | Page::Today => self.render_flat(padded_area, buf),
        }

        IndicatorWidget::new(&IndicatorProps::new(
            self.props.show_more_above,
            self.props.show_more_below,
        ))
        .render(area, buf);
    }
}

/// Builds a styled horizontal divider line with a centered date label.
fn section_line(date: Option<time::Date>, width: usize, color: Color) -> Line<'static> {
    let label = match date {
        Some(date) => format!(" {} ", section_label(date)),
        None => " No Date ".to_string(),
    };
    let label_len = label.chars().count();
    let divider_len = width.saturating_sub(label_len);
    let left_len = divider_len / 2;
    let right_len = divider_len.saturating_sub(left_len);

    Line::from(vec![
        Span::styled(
            "─".repeat(left_len),
            Style::default().fg(color).add_modifier(Modifier::DIM),
        ),
        Span::styled(
            label,
            Style::default().fg(color).add_modifier(Modifier::BOLD),
        ),
        Span::styled(
            "─".repeat(right_len),
            Style::default().fg(color).add_modifier(Modifier::DIM),
        ),
    ])
}

/// Returns a human-friendly label for a date relative to today.
fn section_label(date: time::Date) -> String {
    let current = today();

    if date == current {
        format!("{date} Today")
    } else if date == current + Duration::days(1) {
        format!("{date} Tomorrow")
    } else if date == current - Duration::days(1) {
        format!("{date} Yesterday")
    } else {
        date.to_string()
    }
}
