pub use sea_orm_migration::MigratorTrait;
use sea_orm_migration::prelude::*;

mod m_00_create_todos;
mod m_01_create_sessions;

pub struct Migrator;

#[async_trait::async_trait]
impl MigratorTrait for Migrator {
    fn migrations() -> Vec<Box<dyn MigrationTrait>> {
        vec![
            Box::new(m_00_create_todos::Migration),
            Box::new(m_01_create_sessions::Migration),
        ]
    }
}
