use std::io::Result;

use crate::cmds::Cmd;

pub struct Version;

impl Version {
    pub fn new() -> Self {
        Self {}
    }
}

impl Cmd for Version {
    fn help() -> &'static [&'static str] {
        &["version", "--version", "-V", "Print the current version"]
    }

    fn run(self: Box<Self>) -> Result<()> {
        println!("v{}", env!("CARGO_PKG_VERSION"));
        Ok(())
    }
}
