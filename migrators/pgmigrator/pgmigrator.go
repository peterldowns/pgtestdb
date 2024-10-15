package pgmigrator

import (
	"context"
	"database/sql"
	"io/fs"

	"github.com/peterldowns/pgmigrate"

	"github.com/peterldowns/pgtestdb"

	"github.com/peterldowns/pgtestdb/migrators/common"
)

// Option provides a way to configure the PGMigrator struct and its behavior.
//
// pgmigrate documentation: https://github.com/peterldowns/pgmigrate
//
// See:
//   - [WithTableName]
type Option func(*PGMigrator)

// WithTableName specifies the name of the table in which pgmigrate will store
// its migration records.
//
// Default: `"pgmigrate_migrations"`
func WithTableName(tableName string) Option {
	return func(pgm *PGMigrator) {
		pgm.m.TableName = tableName
	}
}

// WithLogger sets the [pgmigrate.Logger] to use when applying migrations.
//
// You probably want to use `pgmigrate.NewTestLogger(t)`.
//
// Default: `nil`
func WithLogger(logger pgmigrate.Logger) Option {
	return func(pgm *PGMigrator) {
		pgm.m.Logger = logger
	}
}

// New returns a [PGMigrator], which is a pgtestdb.Migrator that uses pgmigrate
// to perform migrations.
func New(dir fs.FS, opts ...Option) (*PGMigrator, error) {
	migrations, err := pgmigrate.Load(dir)
	if err != nil {
		return nil, err
	}
	m := pgmigrate.NewMigrator(migrations)
	pgm := &PGMigrator{m: m}
	for _, opt := range opts {
		opt(pgm)
	}
	return pgm, nil
}

// PGMigrator is a pgtestdb.Migrator that uses pgmigrate to perform migrations.
type PGMigrator struct {
	m *pgmigrate.Migrator
}

func (pgm *PGMigrator) Hash() (string, error) {
	hash := common.NewRecursiveHash(
		common.Field("TableName", pgm.m.TableName),
	)
	for _, migration := range pgm.m.Migrations {
		hash.Add([]byte(migration.SQL))
	}
	return hash.String(), nil
}

func (pgm *PGMigrator) Migrate(
	ctx context.Context,
	db *sql.DB,
	_ pgtestdb.Config,
) error {
	_, err := pgm.m.Migrate(ctx, db)
	return err
}
