use serde::{Deserialize, Serialize};

/// Default name of the GitHub repo `orivo sync` creates/uses for the local database.
fn default_repo_name() -> String {
    "orivo-data".to_string()
}

/// Default name of the synced database blob inside the sync repo.
fn default_file_name() -> String {
    "orivo.sqlite.gz".to_string()
}

/// Configuration for the `sync` command, loaded from the user's config file.
#[derive(Debug, Deserialize, Serialize)]
pub struct SyncConfig {
    /// Name of the GitHub repo, under the signed-in `gh` user's account, used to sync the
    /// local database.
    #[serde(default = "default_repo_name")]
    repo_name: String,
    /// Name of the synced database blob inside the sync repo.
    #[serde(default = "default_file_name")]
    file_name: String,
}

impl Default for SyncConfig {
    /// Returns the built-in default sync configuration.
    fn default() -> Self {
        Self {
            repo_name: default_repo_name(),
            file_name: default_file_name(),
        }
    }
}

impl SyncConfig {
    /// Returns the configured sync repo name.
    pub fn repo_name(&self) -> &str {
        &self.repo_name
    }

    /// Returns the configured sync file name.
    pub fn file_name(&self) -> &str {
        &self.file_name
    }
}
