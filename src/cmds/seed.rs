use std::io::Result;

use crate::{cmds::Cmd, domains::seed::seed_todos, utils::db};

/// Command that resets the database and inserts sample todos.
pub struct Seed;

impl Seed {
    pub fn new() -> Self {
        Self
    }
}

impl Cmd for Seed {
    fn help() -> &'static [&'static str] {
        &["seed", "Seed the database with fake todos for development"]
    }

    fn run(self: Box<Self>) -> Result<()> {
        db::reset()?;
        let db = db::connect()?;
        let inserted = seed_todos(200, &db);
        println!("inserted {inserted} todos");
        Ok(())
    }
}
