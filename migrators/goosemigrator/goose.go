package goosemigrator

import (
	"context"
	"database/sql"
	"io/fs"
	"sync"

	"github.com/pressly/goose/v3"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/common"
)

// The mutex here makes this Migrator concurrency-safe. Goose uses Postgres
// advisory locks as off /v4, which is great, but doesn't prevent issues where
// multiple different tests running in parallel may attempt to run migrations at
// the same time.
var gooseLock sync.Mutex //nolint:gochecknoglobals

// Goose doesn't provide a constant for the default value.
// This will be `"goose_db_version"`.
var DefaultTableName = goose.TableName() //nolint:gochecknoglobals

// Option provides a way to configure the GooseMigrator struct and its behavior.
//
// goose-migrate documentation: https://github.com/pressly/goose#migrations
//
// See:
//   - [WithTableName]
//   - [WithFS]
type Option func(*GooseMigrator)

// WithTableName specifies the name of the table in which goose will store its
// migration records.
//
// Default: `"goose_db_version"`
//
// Equivalent to `-table`
// https://github.com/pressly/goose#usage
func WithTableName(tableName string) Option {
	return func(gm *GooseMigrator) {
		gm.TableName = tableName
	}
}

// WithFS specifies a `fs.FS` from which to read the migration files.
//
// Default: `<nil>` (reads from the real filesystem)
//
// https://github.com/pressly/goose#embedded-sql-migrations
func WithFS(dir fs.FS) Option {
	return func(gm *GooseMigrator) {
		gm.FS = dir
	}
}

// New returns a [GooseMigrator], which is a pgtestdb.Migrator that
// uses goose to perform migrations.
//
// `migrationsDir` is the path to the directory containing migration files.
//
// You can configure the behavior of goose by passing Options:
//   - [WithFS] allows you to use an embedded filesystem.
//   - [WithTableName] is the same as -table
func New(migrationsDir string, opts ...Option) *GooseMigrator {
	gm := &GooseMigrator{
		MigrationsDir: migrationsDir,
		TableName:     DefaultTableName,
	}
	for _, opt := range opts {
		opt(gm)
	}
	return gm
}

// GooseMigrator is a pgtestdb.Migrator that uses goose to perform migrations.
//
// Because Hash() requires calculating a unique hash based on the contents of
// the migrations, database, this implementation only supports reading migration
// files from disk or an embedded filesystem.
//
// GooseMigrator doe snot perform any Verify() or Prepare() logic.
type GooseMigrator struct {
	TableName     string
	MigrationsDir string
	FS            fs.FS
}

func (gm *GooseMigrator) Hash() (string, error) {
	hash := common.NewRecursiveHash(
		common.Field("TableName", gm.TableName),
	)
	if err := hash.AddDirs(gm.FS, "*.sql", gm.MigrationsDir); err != nil {
		return "", err
	}
	return hash.String(), nil
}

// Migrate runs migrate.Up() to migrate the template database.
func (gm *GooseMigrator) Migrate(
	_ context.Context,
	db *sql.DB,
	_ pgtestdb.Config,
) error {
	gooseLock.Lock()
	defer gooseLock.Unlock()
	// Prepare the Goose global state.
	goose.SetBaseFS(gm.FS)
	goose.SetTableName(gm.TableName)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	// Actually runs the migrations.
	return goose.Up(db, gm.MigrationsDir)
}

// Prepare is a no-op method.
func (*GooseMigrator) Prepare(
	_ context.Context,
	_ *sql.DB,
	_ pgtestdb.Config,
) error {
	return nil
}

// Verify is a no-op method.
func (*GooseMigrator) Verify(
	_ context.Context,
	_ *sql.DB,
	_ pgtestdb.Config,
) error {
	return nil
}
