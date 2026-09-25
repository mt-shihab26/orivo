use ratatui::{
    layout::{Constraint, Layout},
    prelude::{Buffer, Color, Rect, Style, Stylize, Widget},
    widgets::{LineGauge, Paragraph},
};

pub struct SessionProps {
    /// Number of completed sessions so far today.
    sessions: u32,
    /// Target number of sessions for the day.
    daily_session_goal: u32,
    /// Phase color.
    color: Color,
}

impl SessionProps {
    pub fn new(sessions: u32, daily_session_goal: u32, color: Color) -> Self {
        Self {
            sessions,
            daily_session_goal,
            color,
        }
    }
}

/// Stateless widget that renders "Session X / Y" text and a centered 50%-wide progress bar.
pub struct SessionWidget<'a> {
    props: &'a SessionProps,
}

impl<'a> SessionWidget<'a> {
    pub fn new(props: &'a SessionProps) -> Self {
        Self { props }
    }
}

impl Widget for &SessionWidget<'_> {
    fn render(self, area: Rect, buf: &mut Buffer) {
        let [text_row, gauge_row] =
            Layout::vertical([Constraint::Length(1), Constraint::Length(1)]).areas(area);

        Paragraph::new(format!(
            "Session {} / {}",
            self.props.sessions, self.props.daily_session_goal
        ))
        .centered()
        .fg(self.props.color)
        .render(text_row, buf);

        let [_, gauge_col, _] = Layout::horizontal([
            Constraint::Fill(1),
            Constraint::Percentage(50),
            Constraint::Fill(1),
        ])
        .areas(gauge_row);

        let ratio = if self.props.daily_session_goal == 0 {
            1.0
        } else {
            (self.props.sessions as f64 / self.props.daily_session_goal as f64).min(1.0)
        };

        let unfilled_color = if self.props.sessions >= self.props.daily_session_goal {
            self.props.color
        } else {
            Color::DarkGray
        };

        LineGauge::default()
            .ratio(ratio)
            .filled_style(Style::new().fg(self.props.color))
            .unfilled_style(Style::new().fg(unfilled_color))
            .render(gauge_col, buf);
    }
}
