//! Binary entrypoint: routes the first CLI argument to a command (default: `tui`).

use std::{
    env,
    io::{Error, ErrorKind, Result},
};

use orivo::{
    cmds::{Cmd, help::Help, sync::Sync, tui::Tui, version::Version},
    config::Config,
    utils::db,
};

fn main() -> Result<()> {
    match env::args().nth(1).as_deref() {
        None | Some("tui") => Box::new(Tui::new(Config::load()?, db::connect()?)).run(),
        #[cfg(debug_assertions)]
        Some("seed") => Box::new(orivo::cmds::seed::Seed::new()).run(),
        Some("sync") => Box::new(Sync::new()).run(),
        Some("version") | Some("--version") | Some("-V") => Box::new(Version::new()).run(),
        Some("help") | Some("--help") | Some("-h") => help(),
        Some(cmd) => unknown(cmd),
    }
}

fn help() -> Result<()> {
    #[cfg(debug_assertions)]
    let helps = [
        Tui::help,
        orivo::cmds::seed::Seed::help,
        Sync::help,
        Version::help,
        Help::help,
    ];
    #[cfg(not(debug_assertions))]
    let helps = [Tui::help, Sync::help, Version::help, Help::help];
    Box::new(Help::new(&helps)).run()
}

fn unknown(unknown: &str) -> Result<()> {
    eprintln!("unknown command: {unknown}");
    eprintln!("run `orivo help` for usage");
    Err(Error::from(ErrorKind::InvalidInput))
}
