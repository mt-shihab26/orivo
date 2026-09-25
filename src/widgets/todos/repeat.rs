use ratatui::{
    crossterm::event::{KeyCode, KeyEvent},
    prelude::{Buffer, Color, Constraint, Layout, Rect, Style, Stylize, Widget},
    widgets::{Block, Borders, List, ListItem, Paragraph},
};

use crate::{kinds::repeat::Repeat, tabs::todos::COLOR};

pub enum RepeatAction {
    /// User confirmed; carries the chosen repeat rule (None means "no repeat").
    Confirm(Option<Repeat>),
    Cancel,
    None,
}

pub struct RepeatProps {
    /// Index of the currently highlighted option (0 = "None", 1+ = Repeat::ALL).
    cursor: usize,
}

impl RepeatProps {
    /// Creates new repeat props, pre-selecting the option matching `selected`.
    pub fn new(selected: Option<&Repeat>) -> Self {
        let cursor = selected
            .and_then(|r| Repeat::ALL.iter().position(|v| v == r).map(|i| i + 1))
            .unwrap_or(0);
        Self { cursor }
    }
}

pub struct RepeatState {
    props: RepeatProps,
}

impl RepeatState {
    pub fn new(props: RepeatProps) -> Self {
        Self { props }
    }

    pub fn props(&self) -> &RepeatProps {
        &self.props
    }

    pub fn handle(&mut self, key: KeyEvent) -> RepeatAction {
        let max = Repeat::ALL.len();
        match key.code {
            KeyCode::Char('j') | KeyCode::Down => {
                self.props.cursor = (self.props.cursor + 1).min(max);
            }
            KeyCode::Char('k') | KeyCode::Up => {
                self.props.cursor = self.props.cursor.saturating_sub(1);
            }
            KeyCode::Enter => {
                let repeat = if self.props.cursor == 0 {
                    None
                } else {
                    Repeat::ALL.get(self.props.cursor - 1).map(Repeat::of)
                };
                return RepeatAction::Confirm(repeat);
            }
            KeyCode::Esc => return RepeatAction::Cancel,
            _ => {}
        }
        RepeatAction::None
    }
}

pub struct RepeatWidget<'a> {
    props: &'a RepeatProps,
}

impl<'a> RepeatWidget<'a> {
    pub fn new(props: &'a RepeatProps) -> Self {
        Self { props }
    }
}

impl Widget for &RepeatWidget<'_> {
    fn render(self, area: Rect, buf: &mut Buffer) {
        let [list_area, hint_area] =
            Layout::vertical([Constraint::Fill(1), Constraint::Length(3)]).areas(area);

        let options = std::iter::once("None").chain(Repeat::ALL.iter().map(|r| r.label()));

        let items: Vec<ListItem> = options
            .enumerate()
            .map(|(i, label)| {
                let active = i == self.props.cursor;
                let style = if active {
                    Style::default().fg(COLOR).bold()
                } else {
                    Style::default().fg(Color::White)
                };
                ListItem::new(format!("{} {}", if active { ">" } else { " " }, label)).style(style)
            })
            .collect();

        List::new(items).render(list_area, buf);

        Paragraph::new(
            "[j/k]Navigate\n\
            [Enter]Confirm\n\
            [Esc]Back",
        )
        .block(
            Block::default()
                .borders(Borders::TOP)
                .border_style(Style::default().fg(COLOR)),
        )
        .fg(Color::DarkGray)
        .render(hint_area, buf);
    }
}
