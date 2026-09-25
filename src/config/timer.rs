use serde::{Deserialize, Serialize};

fn default_work_duration() -> u32 {
    25
}
fn default_break_duration() -> u32 {
    5
}
fn default_long_break_duration() -> u32 {
    15
}
fn default_long_break_interval() -> u32 {
    4
}
fn default_daily_session_goal() -> u32 {
    16
}

/// Configuration for the pomodoro timer, loaded from the user's config file.
#[derive(Debug, Deserialize, Serialize)]
pub struct TimerConfig {
    #[serde(default)]
    show_millis: bool,
    /// Work session duration in minutes.
    #[serde(default = "default_work_duration")]
    work_duration: u32,
    /// Short break duration in minutes.
    #[serde(default = "default_break_duration")]
    break_duration: u32,
    /// Long break duration in minutes.
    #[serde(default = "default_long_break_duration")]
    long_break_duration: u32,
    /// Number of work sessions before a long break.
    #[serde(default = "default_long_break_interval")]
    long_break_interval: u32,
    /// Target number of work sessions to complete each day.
    #[serde(default = "default_daily_session_goal")]
    daily_session_goal: u32,
}

impl Default for TimerConfig {
    fn default() -> Self {
        Self {
            show_millis: false,
            work_duration: default_work_duration(),
            break_duration: default_break_duration(),
            long_break_duration: default_long_break_duration(),
            long_break_interval: default_long_break_interval(),
            daily_session_goal: default_daily_session_goal(),
        }
    }
}

impl TimerConfig {
    pub fn show_millis(&self) -> bool {
        self.show_millis
    }

    /// Work duration in ms, clamped to 1–120 min.
    pub fn work_duration(&self) -> u32 {
        self.work_duration.clamp(1, 120) * 60 * 1000
    }

    /// Short break duration in ms, clamped to 1–60 min.
    pub fn break_duration(&self) -> u32 {
        self.break_duration.clamp(1, 60) * 60 * 1000
    }

    /// Long break duration in ms, clamped to 1–60 min.
    pub fn long_break_duration(&self) -> u32 {
        self.long_break_duration.clamp(1, 60) * 60 * 1000
    }

    /// Work sessions between long breaks, clamped to 1–10.
    pub fn long_break_interval(&self) -> u32 {
        self.long_break_interval.clamp(1, 10)
    }

    /// Daily session goal, clamped to 1–24.
    pub fn daily_session_goal(&self) -> u32 {
        self.daily_session_goal.clamp(1, 24)
    }

    /// Tick interval in ms: 10 when showing millis, 1000 otherwise.
    pub fn tick_interval(show_millis: bool) -> u32 {
        if show_millis { 10 } else { 1000 }
    }
}
