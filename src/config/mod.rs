pub mod sync;
pub mod timer;

use std::{
    fs,
    io::{Error, ErrorKind, Result},
};

use serde::{Deserialize, Serialize};

use crate::{
    config::{sync::SyncConfig, timer::TimerConfig},
    utils::path::config_path,
};

/// Top-level application configuration, loaded from `~/.config/orivo/config.toml`.
#[derive(Debug, Deserialize, Serialize)]
pub struct Config {
    #[serde(default)]
    pub show_fps: bool,
    #[serde(default)]
    pub timer: TimerConfig,
    #[serde(default)]
    pub sync: SyncConfig,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            show_fps: false,
            timer: Default::default(),
            sync: Default::default(),
        }
    }
}

impl Config {
    /// Loads the config from disk, returning the default if the file does not exist.
    pub fn load() -> Result<Self> {
        let path = config_path();
        if !path.exists() {
            return Ok(Self::default());
        }
        let raw = fs::read_to_string(&path)?;
        toml::from_str(&raw).map_err(|e| Error::new(ErrorKind::InvalidData, e.to_string()))
    }
}
