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
	hash := common.NewRecursiveHash(
		common.Field("TableName", sm.MigrationSet.TableName),
		common.Field("SchemaName", sm.MigrationSet.SchemaName),
		// DisableCreateTable is broken for MigrationSet instances, see
		// https://github.com/rubenv/sql-migrate/pull/242 Once that PR is merged
		// and a new release is created, I can bump this repository's
		// dependencies and uncomment this line.
		// common.Field("DisableCreateTable", sm.MigrationSet.DisableCreateTable),
	)
	// Include the contents of the Up migrations.
	for _, migration := range migrations {
		for _, contents := range migration.Up {
			hash.Add([]byte(contents))
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
func (*SQLMigrator) Prepare(
	_ context.Context,
	_ *sql.DB,
	_ testdb.Config,
) error {
	return nil
}

// Verify is a no-op method.
func (*SQLMigrator) Verify(
	_ context.Context,
	_ *sql.DB,
	_ testdb.Config,
) error {
	return nil
}
