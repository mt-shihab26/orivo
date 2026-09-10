use serde::{Deserialize, Serialize};

use crate::log_warn;

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

/// Returns whether `name` is a safe single path/repo segment.
///
/// Both values reach `gh`/`git` as positional arguments and `file_name` is
/// joined onto the sync directory, so an unchecked value is more than a bad
/// name: `../` escapes the sync directory when writing, and a leading `-`
/// is parsed as a flag rather than an argument (`gh repo clone` forwards
/// trailing arguments to `git clone`, which has options that run commands).
fn is_safe_segment(name: &str) -> bool {
    !name.is_empty()
        && name.len() <= 100
        && !name.starts_with('-')
        && name != "."
        && name != ".."
        && name
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || matches!(c, '.' | '_' | '-'))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn accepts_the_defaults() {
        assert!(is_safe_segment("orivo-data"));
        assert!(is_safe_segment("orivo.sqlite.gz"));
    }

    #[test]
    fn rejects_path_traversal() {
        assert!(!is_safe_segment("../../../.bashrc"));
        assert!(!is_safe_segment(".."));
        assert!(!is_safe_segment("."));
        assert!(!is_safe_segment("a/b"));
    }

    #[test]
    fn rejects_values_gh_and_git_would_read_as_flags() {
        assert!(!is_safe_segment("--upload-pack=touch /tmp/pwn"));
        assert!(!is_safe_segment("-c core.sshCommand=id"));
        assert!(!is_safe_segment("-x"));
    }

    #[test]
    fn rejects_empty_overlong_and_shell_metacharacters() {
        assert!(!is_safe_segment(""));
        assert!(!is_safe_segment(&"a".repeat(101)));
        assert!(!is_safe_segment("x;id"));
        assert!(!is_safe_segment("x$(id)"));
        assert!(!is_safe_segment("x y"));
    }

    #[test]
    fn unsafe_values_fall_back_to_the_defaults() {
        let config = SyncConfig {
            repo_name: "../../evil".to_string(),
            file_name: "--upload-pack=id".to_string(),
        };
        assert_eq!(config.repo_name(), "orivo-data");
        assert_eq!(config.file_name(), "orivo.sqlite.gz");
    }
}

impl SyncConfig {
    /// Returns the configured sync repo name, falling back to the default if
    /// the configured value isn't a safe single segment.
    pub fn repo_name(&self) -> &str {
        if is_safe_segment(&self.repo_name) {
            &self.repo_name
        } else {
            log_warn!(
                "config: ignoring unsafe [sync] repo_name {:?}, using the default",
                self.repo_name
            );
            "orivo-data"
        }
    }

    /// Returns the configured sync file name, falling back to the default if
    /// the configured value isn't a safe single segment.
    pub fn file_name(&self) -> &str {
        if is_safe_segment(&self.file_name) {
            &self.file_name
        } else {
            log_warn!(
                "config: ignoring unsafe [sync] file_name {:?}, using the default",
                self.file_name
            );
            "orivo.sqlite.gz"
        }
    }
}
