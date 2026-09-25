pub mod help;
#[cfg(debug_assertions)]
pub mod seed;
pub mod sync;
pub mod tui;
pub mod version;

use std::io::Result;

pub trait Cmd {
    fn run(self: Box<Self>) -> Result<()>;
    fn help() -> &'static [&'static str];
}
