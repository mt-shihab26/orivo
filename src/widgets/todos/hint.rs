use ratatui::{
    prelude::{Buffer, Color, Rect, Stylize, Widget},
    widgets::Paragraph,
};

use crate::kinds::todos_mode::TodosMode;

pub struct HintProps {
    ui_mode: TodosMode,
    /// Whether the delete action is available for the selected todo.
    can_delete: bool,
    /// Whether a search filter is currently active.
    is_searching: bool,
}

impl HintProps {
    pub fn new(ui_mode: TodosMode, can_delete: bool, is_searching: bool) -> Self {
        Self {
            ui_mode,
            can_delete,
            is_searching,
        }
    }
}

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
        let hint = match self.props.ui_mode {
            TodosMode::Normal => match (self.props.can_delete, self.props.is_searching) {
                (true, false) => {
                    "[[/]]Page  [j/k]Navigate  [Space]Toggle  [a]Add  [e]Edit  [^x]Delete  [/]Search"
                }
                (false, false) => {
                    "[[/]]Page  [j/k]Navigate  [Space]Toggle  [a]Add  [e]Edit  [/]Search"
                }
                (true, true) => {
                    "[[/]]Page  [j/k]Navigate  [Space]Toggle  [a]Add  [e]Edit  [^x]Delete  [Esc]Search"
                }
                (false, true) => {
                    "[[/]]Page  [j/k]Navigate  [Space]Toggle  [a]Add  [e]Edit  [Esc]Search"
                }
            },
            TodosMode::Adding | TodosMode::Editing => {
                "[Enter]Confirm  [Esc]Cancel  [Backspace]Delete char"
            }
            TodosMode::Searching => "[Enter]Confirm search  [Esc]Cancel search",
        };

        Paragraph::new(hint)
            .centered()
            .fg(Color::DarkGray)
            .render(area, buf);
    }
}
