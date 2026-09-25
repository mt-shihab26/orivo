use ratatui::{
    crossterm::event::{KeyCode, KeyEvent},
    layout::{Constraint, Layout, Rect},
    prelude::{Buffer, Color, Style, Stylize, Widget},
    widgets::{
        Block, Borders, Clear, Paragraph,
        calendar::{CalendarEventStore, Monthly},
    },
};
use time::{Date, Duration};

use crate::{
    tabs::todos::COLOR,
    utils::date::{shift_month, today},
};

pub enum CalendarAction {
    /// User confirmed; carries the selected date (None = clear date).
    Confirm(Option<Date>),
    Cancel,
    None,
}

pub struct CalendarProps {
    /// Currently highlighted date in the calendar.
    date: Date,
}

impl CalendarProps {
    pub fn new(date: Option<Date>) -> Self {
        Self {
            date: date.unwrap_or_else(today),
        }
    }
}

pub struct CalendarState {
    props: CalendarProps,
}

impl CalendarState {
    pub fn new(props: CalendarProps) -> Self {
        Self { props }
    }

    pub fn props(&self) -> &CalendarProps {
        &self.props
    }

    pub fn handle(&mut self, key: KeyEvent) -> CalendarAction {
        match key.code {
            KeyCode::Char('x') => return CalendarAction::Confirm(None),
            KeyCode::Char('t') => self.navigate(Some(today())),
            KeyCode::Char('y') => self.navigate(today().previous_day()),
            KeyCode::Char('n') => self.navigate(today().next_day()),
            KeyCode::Char('h') | KeyCode::Left => self.navigate(self.props.date.previous_day()),
            KeyCode::Char('l') | KeyCode::Right => self.navigate(self.props.date.next_day()),
            KeyCode::Char('k') | KeyCode::Up => {
                self.navigate(self.props.date.checked_sub(Duration::weeks(1)))
            }
            KeyCode::Char('j') | KeyCode::Down => {
                self.navigate(self.props.date.checked_add(Duration::weeks(1)))
            }
            KeyCode::Char('H') => self.navigate(Some(shift_month(self.props.date, -1))),
            KeyCode::Char('L') => self.navigate(Some(shift_month(self.props.date, 1))),
            KeyCode::Enter => return CalendarAction::Confirm(Some(self.props.date)),
            KeyCode::Esc => return CalendarAction::Cancel,
            _ => {}
        }
        CalendarAction::None
    }

    /// Moves the highlighted date to `date`, ignoring `None` (no-date case).
    fn navigate(&mut self, date: Option<Date>) {
        if let Some(date) = date {
            self.props.date = date;
        }
    }
}

pub struct CalendarWidget<'a> {
    props: &'a CalendarProps,
}

impl<'a> CalendarWidget<'a> {
    pub fn new(props: &'a CalendarProps) -> Self {
        Self { props }
    }
}

impl Widget for &CalendarWidget<'_> {
    fn render(self, area: Rect, buf: &mut Buffer) {
        let popup = centered_rect(area, 24, 5 + 10 + 5);

        Clear.render(popup, buf);

        let block = Block::bordered()
            .title(" Due Date ")
            .border_style(Style::default().fg(COLOR));

        let inner = block.inner(popup);
        block.render(popup, buf);

        let mut events = CalendarEventStore::today(Style::default().fg(Color::Yellow).bold());
        events.add(self.props.date, Style::default().bg(COLOR).fg(Color::Black));

        let [action_hint, cal_area, nav_hint] = Layout::vertical([
            Constraint::Length(5),
            Constraint::Length(10),
            Constraint::Length(5),
        ])
        .areas(inner);

        Paragraph::new(
            "[x]No Date\n\
            [t]Today\n\
            [y]Yesterday\n\
            [n]Tomorrow",
        )
        .block(
            Block::default()
                .borders(Borders::BOTTOM)
                .border_style(Style::default().fg(COLOR)),
        )
        .render(action_hint, buf);

        Monthly::new(self.props.date, events)
            .show_month_header(Style::default().bold())
            .show_weekdays_header(Style::default().fg(Color::DarkGray))
            .render(cal_area, buf);

        Paragraph::new(
            "[h/l]Day\n\
            [j/k]Week\n\
            [H/L]Month\n\
            [Enter]Confirm\n\
            [Esc]Cancel",
        )
        .fg(Color::DarkGray)
        .render(nav_hint, buf);
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
