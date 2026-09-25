use ratatui::{
    prelude::{Buffer, Color, Rect, Stylize, Widget},
    widgets::Paragraph,
};

pub struct HintProps {
    /// Whether the todo-picker overlay is currently open.
    selecting_todo: bool,
    /// Whether the reduce-time dialog is currently open.
    reducing_time: bool,
}

impl HintProps {
    pub fn new(selecting_todo: bool, reducing_time: bool) -> Self {
        Self {
            selecting_todo,
            reducing_time,
        }
    }
}

/// Stateless widget that renders context-sensitive key hints for the timer.
pub struct HintWidget<'a> {
    props: &'a HintProps,
}

impl<'a> HintWidget<'a> {
    pub fn new(props: &'a HintProps) -> Self {
        Self { props }
    }
}

impl Widget for &HintWidget<'_> {
    fn render(self, area: Rect, buf: &mut Buffer) {
        let text = if self.props.selecting_todo {
            "[j/k] Navigate   [Enter] Select   [Esc] Cancel"
        } else if self.props.reducing_time {
            "[0-9] Enter time   [Backspace] Delete   [Enter] Apply   [Esc] Cancel"
        } else {
            "[Space] Toggle   [r] Reset   [n] Skip   [t] Todo   [T] Clear   [m] Millis   [d] Reduce"
        };
        Paragraph::new(text)
            .centered()
            .fg(Color::DarkGray)
            .render(area, buf);
    }
}
