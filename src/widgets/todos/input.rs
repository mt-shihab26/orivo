use ratatui::{
    crossterm::event::{KeyCode, KeyEvent, KeyModifiers},
    prelude::{Buffer, Color, Constraint, Layout, Rect, Style, Stylize, Widget},
    text::Span,
    widgets::{Block, Clear, Paragraph},
};
use ratatui_textarea::TextArea;
use time::OffsetDateTime;

use crate::{kinds::repeat::Repeat, tabs::todos::COLOR, utils::date::now};

use super::{
    calendar::{CalendarAction, CalendarProps, CalendarState, CalendarWidget},
    repeat::{RepeatAction, RepeatProps, RepeatState, RepeatWidget},
};

/// Action returned by the input widget after handling a key event.
pub enum InputAction {
    /// User submitted the input; carries the text, optional due date, and repeat rule.
    Confirm {
        /// The trimmed text entered by the user.
        text: String,
        /// Optional due date chosen via the calendar picker.
        date: Option<OffsetDateTime>,
        /// Optional repeat rule chosen via the repeat picker.
        repeat: Option<Repeat>,
    },
    /// User pressed Escape to cancel input.
    Escape,
    /// No state change occurred.
    None,
}

/// Props for the todo text-input widget.
pub struct InputProps {
    /// The textarea holding the user's current text input.
    textarea: TextArea<'static>,
    /// Currently selected due date, if any.
    date: Option<OffsetDateTime>,
    /// Currently selected repeat rule, if any.
    repeat: Option<Repeat>,
}

impl InputProps {
    /// Creates new input props, optionally pre-filling text, date, and repeat.
    pub fn new(text: Option<&str>, date: Option<OffsetDateTime>, repeat: Option<&Repeat>) -> Self {
        let mut textarea = TextArea::default();
        if let Some(t) = text {
            textarea.insert_str(t);
        }
        textarea.set_block(Block::bordered().border_style(Style::default().fg(COLOR)));
        textarea.set_cursor_line_style(Style::default());
        Self {
            textarea,
            date,
            repeat: repeat.map(Repeat::of),
        }
    }
}

/// Stateful container for the todo input, owns props and optional picker overlays.
pub struct InputState {
    /// Mutable props updated as the user types or picks a date/repeat.
    props: InputProps,
    /// Active calendar state when the date-picker overlay is open.
    calendar_state: Option<CalendarState>,
    /// Active repeat state when the repeat-picker overlay is open.
    repeat_state: Option<RepeatState>,
}

impl InputState {
    /// Creates a new input state wrapping the given props.
    pub fn new(props: InputProps) -> Self {
        Self {
            props,
            calendar_state: None,
            repeat_state: None,
        }
    }

    /// Returns a shared reference to the current props.
    pub fn props(&self) -> &InputProps {
        &self.props
    }

    /// Handles a key event, delegating to the calendar or repeat overlay when open.
    pub fn handle(&mut self, key: KeyEvent) -> InputAction {
        if let Some(cal) = &mut self.calendar_state {
            match cal.handle(key) {
                CalendarAction::Confirm { date } => {
                    self.props.date = date.map(|d| now().replace_date(d));
                    self.calendar_state = None;
                }
                CalendarAction::Cancel => self.calendar_state = None,
                CalendarAction::None => {}
            }
            return InputAction::None;
        }

        if let Some(rep) = &mut self.repeat_state {
            match rep.handle(key) {
                RepeatAction::Confirm(repeat) => {
                    self.props.repeat = repeat;
                    self.repeat_state = None;
                }
                RepeatAction::Cancel => self.repeat_state = None,
                RepeatAction::None => {}
            }
            return InputAction::None;
        }

        match key.code {
            KeyCode::Enter => {
                let text = self.props.textarea.lines()[0].clone();
                if !text.trim().is_empty() {
                    return InputAction::Confirm {
                        text,
                        date: self.props.date,
                        repeat: self.props.repeat.as_ref().map(Repeat::of),
                    };
                }
            }
            KeyCode::Esc => return InputAction::Escape,
            KeyCode::Char('d') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                self.calendar_state = Some(CalendarState::new(CalendarProps::new(
                    self.props.date.map(|dt| dt.date()),
                )));
            }
            KeyCode::Char('e') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                self.repeat_state = Some(RepeatState::new(RepeatProps::new(self.props.repeat.as_ref())));
            }
            _ => {
                self.props.textarea.input(key);
            }
        }
        InputAction::None
    }

    /// Renders the calendar overlay into the buffer if it is currently open.
    pub fn render_calendar(&self, area: Rect, buf: &mut Buffer) {
        if let Some(cal) = &self.calendar_state {
            CalendarWidget::new(cal.props()).render(area, buf);
        }
    }

    /// Renders the repeat-picker overlay into the buffer if it is currently open.
    pub fn render_repeat(&self, area: Rect, buf: &mut Buffer) {
        let Some(rep) = &self.repeat_state else { return };

        let popup_w = 26u16;
        let popup_h = 14u16;
        let popup = ratatui::layout::Rect {
            x: area.x + area.width.saturating_sub(popup_w) / 2,
            y: area.y + area.height.saturating_sub(popup_h) / 2,
            width: popup_w.min(area.width),
            height: popup_h.min(area.height),
        };

        Clear.render(popup, buf);

        let block = Block::bordered()
            .title(" Repeat ")
            .border_style(Style::default().fg(COLOR));
        let inner = block.inner(popup);
        block.render(popup, buf);

        RepeatWidget::new(rep.props()).render(inner, buf);
    }
}

/// Stateless widget that renders the todo text-input row.
pub struct InputWidget<'a> {
    /// Borrowed input props for this render pass.
    props: &'a InputProps,
}

impl<'a> InputWidget<'a> {
    /// Creates a new input widget from the given props.
    pub fn new(props: &'a InputProps) -> Self {
        Self { props }
    }
}

impl Widget for &InputWidget<'_> {
    /// Renders the textarea alongside separate date and repeat blocks.
    fn render(self, area: Rect, buf: &mut Buffer) {
        let repeat_width = self
            .props
            .repeat
            .as_ref()
            .map(|r| (r.label().len() as u16 + 4).max(14))
            .unwrap_or(14);

        let [text_area, date_area, repeat_area] = Layout::horizontal([
            Constraint::Fill(1),
            Constraint::Length(14),
            Constraint::Length(repeat_width),
        ])
        .areas(area);

        Widget::render(&self.props.textarea, text_area, buf);

        let date_str = match self.props.date {
            Some(d) => format!("{}", d.date()),
            None => "no date".to_string(),
        };
        let date_block = Block::bordered()
            .title(Span::from(" ^d ").fg(Color::DarkGray).bold())
            .border_style(Style::default().fg(COLOR));
        let date_inner = date_block.inner(date_area);
        date_block.render(date_area, buf);
        Paragraph::new(date_str).fg(COLOR).centered().render(date_inner, buf);

        let repeat_str = self
            .props
            .repeat
            .as_ref()
            .map(|r| r.label().to_string())
            .unwrap_or_else(|| "no repeat".to_string());
        let repeat_block = Block::bordered()
            .title(Span::from(" ^e ").fg(Color::DarkGray).bold())
            .border_style(Style::default().fg(COLOR));
        let repeat_inner = repeat_block.inner(repeat_area);
        repeat_block.render(repeat_area, buf);
        Paragraph::new(repeat_str)
            .fg(COLOR)
            .centered()
            .render(repeat_inner, buf);
    }
}
