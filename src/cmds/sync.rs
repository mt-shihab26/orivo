use std::io::Result;

use crate::{cmds::Cmd, domains::sync::run_sync};

pub struct Sync;

impl Sync {
    pub fn new() -> Self {
        Self
    }
}

impl Cmd for Sync {
    fn help() -> &'static [&'static str] {
        &[
            "sync",
            "Sync your data with GitHub (pulls and/or pushes as needed)",
        ]
    }

    fn run(self: Box<Self>) -> Result<()> {
        run_sync()
    }
}
