use std::{sync::mpsc::Sender, thread};

use ratatui::crossterm::event::{self, Event as TerminalEvent, KeyEventKind};

use crate::{kinds::event::Event, log_error, log_warn};

/// Spawns a thread that forwards key presses, mouse, and resize events to the main loop.
pub fn spawn(sender: Sender<Event>) {
    thread::spawn(move || {
        loop {
            match event::read() {
                Err(e) => {
                    log_warn!("term worker: failed to read terminal event: {e}");
                }
                Ok(TerminalEvent::Key(key)) => {
                    if key.kind == KeyEventKind::Press {
                        if sender.send(Event::Key(key)).is_err() {
                            log_error!("term worker: event channel closed, stopping");
                            break;
                        }
                    }
                }
                Ok(TerminalEvent::Mouse(mouse)) => {
                    if sender.send(Event::Mouse(mouse)).is_err() {
                        log_error!("term worker: event channel closed, stopping");
                        break;
                    }
                }
                Ok(TerminalEvent::Resize(width, height)) => {
                    if sender.send(Event::Resize(width, height)).is_err() {
                        log_error!("term worker: event channel closed, stopping");
                        break;
                    }
                }
                Ok(_) => {}
            }
        }
    });
}
