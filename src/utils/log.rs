use std::{
    fs::{self, OpenOptions},
    io::Write,
    time::SystemTime,
};

use crate::utils::path::log_path;

/// Returns the current UTC time as an ISO-8601 string (no external deps).
fn timestamp() -> String {
    let secs = SystemTime::now()
        .duration_since(SystemTime::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0);

    // manual UTC breakdown from unix epoch
    let s = secs % 60;
    let m = (secs / 60) % 60;
    let h = (secs / 3600) % 24;

    let days = secs / 86400;
    let (mut year, mut day) = (1970u64, days);

    loop {
        let days_in_year = if is_leap(year) { 366 } else { 365 };
        if day < days_in_year {
            break;
        }
        day -= days_in_year;
        year += 1;
    }

    let months = [
        31u64,
        if is_leap(year) { 29 } else { 28 },
        31,
        30,
        31,
        30,
        31,
        31,
        30,
        31,
        30,
        31,
    ];
    let mut month = 1u64;
    for days_in_month in months {
        if day < days_in_month {
            break;
        }
        day -= days_in_month;
        month += 1;
    }

    format!("{year}-{month:02}-{day:02}T{h:02}:{m:02}:{s:02}Z")
}

/// Returns `true` if `year` is a Gregorian leap year.
fn is_leap(year: u64) -> bool {
    (year % 4 == 0 && year % 100 != 0) || year % 400 == 0
}

/// Size at which the log is rotated. Every query is logged on every render
/// tick, so without a bound this grows for as long as the app is installed.
const MAX_LOG_BYTES: u64 = 5 * 1024 * 1024;

/// Appends a timestamped log line with the given level and message to the log file.
pub fn write(level: &str, msg: &str) {
    let path = log_path();
    if let Some(parent) = path.parent() {
        let _ = fs::create_dir_all(parent);
    }

    // Keep one previous log around, so rotating never loses the most recent
    // history outright — the older file is only dropped a rotation later.
    if fs::metadata(&path).map(|m| m.len()).unwrap_or(0) >= MAX_LOG_BYTES {
        let _ = fs::rename(&path, path.with_extension("log.1"));
    }

    if let Ok(mut file) = OpenOptions::new().create(true).append(true).open(&path) {
        let _ = writeln!(file, "[{}] {}: {}", timestamp(), level, msg);
    }
}

/// `log::Log` implementation that routes sqlx queries to the app log file.
struct DbLogger;

impl log::Log for DbLogger {
    /// Returns `true` only for sqlx targets at Info level or below.
    fn enabled(&self, metadata: &log::Metadata) -> bool {
        metadata.target().starts_with("sqlx") && metadata.level() <= log::Level::Info
    }

    /// Writes the log record to the app log file if enabled.
    fn log(&self, record: &log::Record) {
        if self.enabled(record.metadata()) {
            write("DB", &format!("{}", record.args()));
        }
    }

    /// No-op: log lines are written synchronously on each call.
    fn flush(&self) {}
}

static LOGGER: DbLogger = DbLogger;

/// Registers `DbLogger` as the global logger and sets the max level to Info.
pub fn init() {
    let _ = log::set_logger(&LOGGER);
    log::set_max_level(log::LevelFilter::Info);
}

/// Logs a message at the ERROR level to the app log file.
#[macro_export]
macro_rules! log_error {
    ($($arg:tt)*) => { $crate::utils::log::write("ERROR", &format!($($arg)*)) };
}

/// Logs a message at the WARN level to the app log file.
#[macro_export]
macro_rules! log_warn {
    ($($arg:tt)*) => { $crate::utils::log::write("WARN", &format!($($arg)*)) };
}

/// Logs a message at the INFO level to the app log file.
#[macro_export]
macro_rules! log_info {
    ($($arg:tt)*) => { $crate::utils::log::write("INFO", &format!($($arg)*)) };
}
