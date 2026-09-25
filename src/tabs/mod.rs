pub mod timer;
pub mod todos;

use std::io::Result;

use ratatui::{Frame, crossterm::event::KeyEvent, layout::Rect, style::Color};

pub trait Tab {
    fn name(&self) -> &str;
    fn color(&self) -> Color;
    fn handle(&mut self, key: KeyEvent) -> Result<()>;
    fn render(&self, frame: &mut Frame, area: Rect);
    /// Returns `true` if this tab needs periodic tick events; defaults to `false`.
    fn should_tick(&self) -> bool {
        false
    }
    /// Called on each tick when `should_tick` returns `true`; defaults to no-op.
    fn next_tick(&mut self) -> Result<()> {
        Ok(())
    }
    /// Drops any cached data held by this tab; defaults to no-op.
    fn invalidate_cache(&mut self) {}
}
