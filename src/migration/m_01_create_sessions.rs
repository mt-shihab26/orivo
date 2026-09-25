use sea_orm_migration::prelude::*;

pub struct Migration;

#[derive(DeriveIden)]
enum Sessions {
    Table,
    Id,
    Phase,
    DurationSecs,
    StartedAt,
    EndedAt,
    TodoId,
    CreatedAt,
    UpdatedAt,
}

impl MigrationName for Migration {
    fn name(&self) -> &str {
        "m_01_create_sessions"
    }
}

#[async_trait::async_trait]
impl MigrationTrait for Migration {
    async fn up(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .create_table(
                Table::create()
                    .table(Sessions::Table)
                    .if_not_exists()
                    .col(
                        ColumnDef::new(Sessions::Id)
                            .integer()
                            .not_null()
                            .auto_increment()
                            .primary_key(),
                    )
                    .col(ColumnDef::new(Sessions::Phase).string().not_null())
                    .col(ColumnDef::new(Sessions::DurationSecs).integer().not_null())
                    .col(ColumnDef::new(Sessions::StartedAt).string().null())
                    .col(ColumnDef::new(Sessions::EndedAt).string().null())
                    .col(ColumnDef::new(Sessions::TodoId).integer().null())
                    .col(
                        ColumnDef::new(Sessions::CreatedAt)
                            .string()
                            .not_null()
                            .default(""),
                    )
                    .col(
                        ColumnDef::new(Sessions::UpdatedAt)
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
            .drop_table(Table::drop().table(Sessions::Table).to_owned())
            .await
    }
}
