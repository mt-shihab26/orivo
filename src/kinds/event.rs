use ratatui::crossterm::event::{KeyEvent, MouseEvent};

pub enum Event {
    Key(KeyEvent),
    Mouse(MouseEvent),
    /// New terminal width and height.
    Resize(u16, u16),
    /// A periodic timer tick emitted by the timer worker.
    TimerTick,
}
