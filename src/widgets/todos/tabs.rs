use ratatui::{
    prelude::{Buffer, Color, Constraint, Layout, Rect, Style, Widget},
    widgets::Tabs,
};

use crate::kinds::page::Page;

pub struct TabsProps {
    page: Page,
    color: Color,
}

impl TabsProps {
    pub fn new(page: Page, color: Color) -> Self {
        Self { page, color }
    }
}

pub struct TabsWidget<'a> {
    props: &'a TabsProps,
}

impl<'a> TabsWidget<'a> {
    pub fn new(props: &'a TabsProps) -> Self {
        Self { props }
    }
}

impl Widget for &TabsWidget<'_> {
    fn render(self, area: Rect, buf: &mut Buffer) {
        let tab_titles: Vec<&str> = Page::ALL.iter().map(|p| p.label()).collect();
        let tabs_width: u16 = Page::ALL
            .iter()
            .map(|p| p.label().len() as u16 + 2)
            .sum::<u16>()
            + (Page::ALL.len() as u16 - 1) * 3;

        let [area, _] =
            Layout::horizontal([Constraint::Length(tabs_width), Constraint::Fill(1)]).areas(area);

        Tabs::new(tab_titles)
            .select(self.props.page.index())
            .style(Style::default().fg(Color::DarkGray))
            .highlight_style(Style::default().fg(self.props.color).bold())
            .divider(" | ")
            .render(area, buf);
    }
}
