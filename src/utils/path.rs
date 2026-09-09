use std::{env, path::PathBuf};

/// Returns the path to the config file.
pub fn config_path() -> PathBuf {
    config_base_path().join("config.toml")
}

/// Returns the path to the log file.
pub fn log_path() -> PathBuf {
    state_base_path().join("orivo.log")
}

/// Returns the path to the database file.
pub fn db_path() -> PathBuf {
    state_base_path().join("orivo.sqlite")
}

/// Returns the path to the persisted timer state file.
pub fn store_path() -> PathBuf {
    state_base_path().join("store.json")
}

/// Returns the path to the Unix domain socket that exposes live timer state
/// (including `is_running`, which is never persisted to `store.json`) to
/// external tools such as bar widgets.
pub fn ipc_socket_path() -> PathBuf {
    state_base_path().join("orivo.sock")
}

/// Returns the path to the persisted sync state file (hash of the database as of the last
/// successful `orivo sync`).
pub fn sync_state_path() -> PathBuf {
    state_base_path().join("sync.json")
}

/// Returns the local git working copy used to sync the database with GitHub.
pub fn sync_dir() -> PathBuf {
    state_base_path().join("sync")
}

/// The base directory for config files.
#[cfg_attr(debug_assertions, allow(dead_code))]
fn config_base_path() -> PathBuf {
    #[cfg(debug_assertions)]
    return local();

    #[cfg(not(debug_assertions))]
    PathBuf::from(home())
        .join(".config")
        .join(env!("CARGO_PKG_NAME"))
}

/// the base directory for runtime state files
#[cfg_attr(debug_assertions, allow(dead_code))]
fn state_base_path() -> PathBuf {
    #[cfg(debug_assertions)]
    return local();

    #[cfg(not(debug_assertions))]
    PathBuf::from(home())
        .join(".local")
        .join("state")
        .join(env!("CARGO_PKG_NAME"))
}

/// Returns the user's home directory, handling both Unix (`HOME`) and Windows (`USERPROFILE`).
#[cfg_attr(debug_assertions, allow(dead_code))]
fn home() -> String {
    env::var("HOME")
        .or_else(|_| env::var("USERPROFILE"))
        .unwrap_or_else(|_| ".".to_string())
}

/// Returns the local dev directory used for all files in debug builds.
#[cfg(debug_assertions)]
fn local() -> PathBuf {
    PathBuf::from("./.dev")
}
