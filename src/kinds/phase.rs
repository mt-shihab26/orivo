use ratatui::style::Color;
use serde::{Deserialize, Serialize};

use crate::config::timer::TimerConfig;

/// A phase in the pomodoro cycle.
#[derive(Debug, Clone, Deserialize, Serialize)]
pub enum Phase {
    Work,
    Break,
    /// A longer break after completing a full interval of work sessions.
    LongBreak,
}

impl Phase {
    pub fn label(&self) -> &str {
        match self {
            Phase::Work => "Work Session",
            Phase::Break => "Short Break",
            Phase::LongBreak => "Long Break",
        }
    }

    pub fn color(&self) -> Color {
        match self {
            Phase::Work => Color::Red,
            Phase::Break => Color::Green,
            Phase::LongBreak => Color::Cyan,
        }
    }

    pub fn to_db_str(&self) -> &str {
        match self {
            Phase::Work => "work",
            Phase::Break => "break",
            Phase::LongBreak => "long_break",
        }
    }

    pub fn from_db_str(s: &str) -> Option<Phase> {
        match s {
            "work" => Some(Phase::Work),
            "break" => Some(Phase::Break),
            "long_break" => Some(Phase::LongBreak),
            _ => None,
        }
    }

    /// Configured duration in milliseconds.
    pub fn duration(&self, timer_config: &TimerConfig) -> u32 {
        match self {
            Phase::Work => timer_config.work_duration(),
            Phase::Break => timer_config.break_duration(),
            Phase::LongBreak => timer_config.long_break_duration(),
        }
    }
}
