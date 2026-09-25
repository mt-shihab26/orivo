use ratatui::{
    prelude::{Buffer, Color, Rect, Stylize, Widget},
    widgets::Paragraph,
};

pub struct StatusProps {
    running: bool,
    /// Phase color.
    color: Color,
}

impl StatusProps {
    pub fn new(running: bool, color: Color) -> Self {
        Self { running, color }
    }
}

/// Stateless widget that renders "Running" or "Paused".
pub struct StatusWidget<'a> {
    props: &'a StatusProps,
}

impl<'a> StatusWidget<'a> {
    pub fn new(props: &'a StatusProps) -> Self {
        Self { props }
    }
}

impl Widget for &StatusWidget<'_> {
    fn render(self, area: Rect, buf: &mut Buffer) {
        let (label, color) = if self.props.running {
            ("Running", self.props.color)
        } else {
            ("Paused", Color::DarkGray)
        };
        Paragraph::new(label).centered().fg(color).render(area, buf);
    }
}
