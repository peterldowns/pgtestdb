package sqlmigrator

import (
	"context"
	"database/sql"

	migrate "github.com/rubenv/sql-migrate"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/common"
)

// New returns a [SQLMigrator], which is a pgtestdb.Migrator that uses sql-migrate
// to perform migrations.
//
// `source` and `migrationSet` are both types defined by sql-migrate, for more
// information see their documentation.
//
// https://github.com/rubenv/sql-migrate#as-a-library
func New(source migrate.MigrationSource, migrationSet *migrate.MigrationSet) *SQLMigrator {
	if migrationSet == nil {
		migrationSet = &migrate.MigrationSet{}
	}
	return &SQLMigrator{
		MigrationSet: migrationSet,
		Source:       source,
	}
}

// SQLMigrator is a pgtestdb.Migrator that uses sql-migrate to perform migrations.
type SQLMigrator struct {
	Source       migrate.MigrationSource
	MigrationSet *migrate.MigrationSet
}

func (sm *SQLMigrator) Hash() (string, error) {
	migrations, err := sm.Source.FindMigrations()
	if err != nil {
		return "", err
	}
	// Include settings/values in the hash that affect the resulting schema of
	// the database,
	hash := common.NewRecursiveHash(
		common.Field("TableName", sm.MigrationSet.TableName),
		common.Field("SchemaName", sm.MigrationSet.SchemaName),
	)
	// Include the contents of the Up migrations.
	for _, migration := range migrations {
		for _, contents := range migration.Up {
			hash.Add([]byte(contents))
		}
	}
	return hash.String(), nil
}

// Migrate runs migrationSet.Exec() to migrate the template database.
func (sm *SQLMigrator) Migrate(
	_ context.Context,
	db *sql.DB,
	_ pgtestdb.Config,
) error {
	_, err := sm.MigrationSet.Exec(db, "postgres", sm.Source, migrate.Up)
	return err
}
