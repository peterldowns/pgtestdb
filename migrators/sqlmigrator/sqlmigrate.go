package sqlmigrator

import (
	"context"
	"database/sql"

	migrate "github.com/rubenv/sql-migrate"

	"github.com/peterldowns/testdb"
	"github.com/peterldowns/testdb/migrators/common"
)

func New(source migrate.MigrationSource, migrationSet *migrate.MigrationSet) *SQLMigrator {
	if migrationSet == nil {
		migrationSet = &migrate.MigrationSet{}
	}
	return &SQLMigrator{
		MigrationSet: migrationSet,
		Source:       source,
	}
}

type SQLMigrator struct {
	Source       migrate.MigrationSource
	MigrationSet *migrate.MigrationSet
}

func (sm *SQLMigrator) Hash() (string, error) {
	migrations, err := sm.Source.FindMigrations()
	if err != nil {
		return "", err
	}
	// Include settings/values in the hash that affect the resulting schema of the database,
	hash, err := common.NewRecursiveHash(
		common.Field("TableName", sm.MigrationSet.TableName),
		common.Field("SchemaName", sm.MigrationSet.SchemaName),
		common.Field("DisableCreateTable", sm.MigrationSet.DisableCreateTable),
	)
	if err != nil {
		return "", err
	}
	// Include the contents of the Up migrations.
	for _, migration := range migrations {
		for _, contents := range migration.Up {
			err := hash.Add([]byte(contents))
			if err != nil {
				return "", err
			}
		}
	}
	return hash.String(), nil
}

func (sm *SQLMigrator) Migrate(
	_ context.Context,
	db *sql.DB,
	_ testdb.Config,
) error {
	_, err := sm.MigrationSet.Exec(db, "postgres", sm.Source, migrate.Up)
	return err
}

// Prepare is a no-op method.
func (sm *SQLMigrator) Prepare(
	_ context.Context,
	_ *sql.DB,
	_ testdb.Config,
) error {
	return nil
}

// Verify is a no-op method.
func (sm *SQLMigrator) Verify(
	_ context.Context,
	_ *sql.DB,
	_ testdb.Config,
) error {
	return nil
}
