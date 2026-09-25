use std::io::Result;

use sea_orm::DatabaseConnection;

use crate::{config::Config, domains::tui::App};

use super::Cmd;

pub struct Tui {
    config: Config,
    db: DatabaseConnection,
}

impl Tui {
    pub fn new(config: Config, db: DatabaseConnection) -> Self {
        Self { config, db }
    }
}

impl Cmd for Tui {
    fn help() -> &'static [&'static str] {
        &["(default)", "tui", "Launch the terminal UI"]
    }

    fn run(self: Box<Self>) -> Result<()> {
        let mut app = App::new(self.config, self.db);

        app.run()
    }
}
