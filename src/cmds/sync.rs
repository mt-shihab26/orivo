use std::io::Result;

use crate::{cmds::Cmd, domains::sync::run_sync};

/// Command that syncs the local database with the user's `orivo-data` GitHub repo, creating
/// it via the github cli if it doesn't exist yet.
pub struct Sync;

impl Sync {
    /// Creates a new `Sync` command.
    pub fn new() -> Self {
        Self
    }
}

impl Cmd for Sync {
    /// Returns the CLI name and description for the sync command.
    fn help() -> &'static [&'static str] {
        &[
            "sync",
            "Sync your data with GitHub (pulls and/or pushes as needed)",
        ]
    }

    /// Pulls and/or pushes the local database against the GitHub sync repo as needed.
    fn run(self: Box<Self>) -> Result<()> {
        run_sync()
    }
}
