use sea_orm_migration::prelude::*;

pub struct Migration;

#[derive(DeriveIden)]
enum Todos {
    Table,
    Id,
    Text,
    DoneAt,
    DueDate,
    Repeat,
    ParentId,
    CreatedAt,
    UpdatedAt,
}

impl MigrationName for Migration {
    fn name(&self) -> &str {
        "m_00_create_todos"
    }
}

#[async_trait::async_trait]
impl MigrationTrait for Migration {
    async fn up(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .create_table(
                Table::create()
                    .table(Todos::Table)
                    .if_not_exists()
                    .col(
                        ColumnDef::new(Todos::Id)
                            .integer()
                            .not_null()
                            .auto_increment()
                            .primary_key(),
                    )
                    .col(ColumnDef::new(Todos::Text).string().not_null())
                    .col(ColumnDef::new(Todos::DoneAt).string().null())
                    .col(ColumnDef::new(Todos::DueDate).string().null())
                    .col(ColumnDef::new(Todos::Repeat).string().null())
                    .col(ColumnDef::new(Todos::ParentId).integer().null())
                    .col(
                        ColumnDef::new(Todos::CreatedAt)
                            .string()
                            .not_null()
                            .default(""),
                    )
                    .col(
                        ColumnDef::new(Todos::UpdatedAt)
                            .string()
                            .not_null()
                            .default(""),
                    )
                    .to_owned(),
            )
            .await
    }

    async fn down(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .drop_table(Table::drop().table(Todos::Table).to_owned())
            .await
    }
}
