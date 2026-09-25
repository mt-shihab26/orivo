use std::{io::Result, sync::OnceLock};

use sea_orm::{ConnectOptions, Database, DatabaseConnection};

use crate::{
    migration::{Migrator, MigratorTrait},
    utils::path::db_path,
};

static RUNTIME: OnceLock<tokio::runtime::Runtime> = OnceLock::new();

/// Returns the global Tokio runtime, initializing it on first call.
pub fn rt() -> &'static tokio::runtime::Runtime {
    RUNTIME.get_or_init(|| {
        tokio::runtime::Builder::new_multi_thread()
            .enable_all()
            .build()
            .expect("failed to build tokio runtime")
    })
}

/// Opens (or creates) the SQLite database and runs any pending migrations.
pub fn connect() -> Result<DatabaseConnection> {
    let path = db_path();
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent)?;
    }

    crate::utils::log::init();

    rt().block_on(async {
        let url = format!("sqlite://{}?mode=rwc", path.to_str().unwrap());

        let mut opts = ConnectOptions::new(url);
        opts.sqlx_logging(true)
            .sqlx_logging_level(log::LevelFilter::Info);

        let db = Database::connect(opts).await.map_err(io_err)?;

        Migrator::up(&db, None).await.map_err(io_err)?;

        Ok(db)
    })
}

/// Deletes the database file, effectively resetting all stored data.
pub fn reset() -> Result<()> {
    let path = db_path();
    if path.exists() {
        std::fs::remove_file(path)?;
    }
    Ok(())
}

fn io_err(e: impl std::fmt::Display) -> std::io::Error {
    std::io::Error::new(std::io::ErrorKind::Other, e.to_string())
}
