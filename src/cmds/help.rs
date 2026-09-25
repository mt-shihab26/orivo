use std::io::Result;

use super::Cmd;

pub struct Help {
    lines: Vec<Vec<String>>,
}

impl Help {
    pub fn new(helps: &[fn() -> &'static [&'static str]]) -> Self {
        let lines = helps
            .iter()
            .map(|help| help().iter().map(|s| s.to_string()).collect())
            .collect();

        Self { lines }
    }

    fn format(entries: &[&str]) -> String {
        let (description, names) = entries.split_last().unwrap_or((&"", &[]));
        format!("{:<30} {}", names.join(", "), description)
    }
}

impl Cmd for Help {
    fn help() -> &'static [&'static str] {
        &["help", "--help", "-h", "Show help for all commands"]
    }

    fn run(self: Box<Self>) -> Result<()> {
        println!("Usage: {} <command>\n", env!("CARGO_PKG_NAME"));
        println!("Commands:");
        for entries in &self.lines {
            let strs: Vec<&str> = entries.iter().map(|s| s.as_str()).collect();
            println!("  {}", Self::format(&strs));
        }
        Ok(())
    }
}
