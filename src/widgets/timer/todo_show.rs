use ratatui::{
    prelude::{Buffer, Color, Rect, Stylize, Widget},
    widgets::Paragraph,
};

use crate::models::{session::Stat, todo::Todo};

/// Props for the active-todo display bar at the bottom of the timer.
pub struct TodoShowProps<'a> {
    /// The todo currently linked to the timer session, if any.
    todo: Option<&'a Todo>,
    /// Accumulated session statistics for the active todo, if any.
    stat: Option<&'a Stat>,
    /// Phase color.
    color: Color,
}

impl<'a> TodoShowProps<'a> {
    pub fn new(todo: Option<&'a Todo>, stat: Option<&'a Stat>, color: Color) -> Self {
        Self { todo, stat, color }
    }
}

pub struct TodoShowWidget<'a> {
    props: &'a TodoShowProps<'a>,
}

impl<'a> TodoShowWidget<'a> {
    pub fn new(props: &'a TodoShowProps<'a>) -> Self {
        Self { props }
    }
}

impl Widget for &TodoShowWidget<'_> {
    fn render(self, area: Rect, buf: &mut Buffer) {
        let text = match self.props.todo {
            Some(todo) => match self.props.stat {
                Some(stat) => {
                    format!(
                        "{}  ·  {} sessions  ·  {} min",
                        todo.text,
                        stat.completed_sessions,
                        stat.completed_secs / 60
                    )
                }
                None => todo.text.clone(),
            },
            None => "No todo selected  [t] pick".to_string(),
        };
        Paragraph::new(text)
            .centered()
            .fg(self.props.color)
            .render(area, buf);
    }
}
