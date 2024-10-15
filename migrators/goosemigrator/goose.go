package goosemigrator

import (
	"context"
	"database/sql"
	"io/fs"
	"os"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/common"
)

// Goose doesn't provide a constant for the default value.
// This will be `"goose_db_version"`.
var DefaultTableName = goose.DefaultTablename //nolint:gochecknoglobals

// Option provides a way to configure the `goose.Provider` used by [GooseMigrator] to
// run migrations.
//
// goose-migrate documentation:
// - https://github.com/pressly/goose#migrations
// - https://pressly.github.io/goose/documentation/provider/
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
// Default: `os.DirFS(".")` (reads from the real filesystem in the current working directory)
//
// https://github.com/pressly/goose#embedded-sql-migrations
func WithFS(dir fs.FS) Option {
	return func(gm *GooseMigrator) {
		gm.FS = dir
	}
}

// New returns a [GooseMigrator], which is a pgtestdb.Migrator that creates a
// `goose.Provider` to perform migrations. It is limited in functionality and
// does not allow you to configure the full range of `goose.ProviderOptions`.
// If you need a more complicated implementation, please write your own (and
// consider contributing it back to this project!)
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
		FS:            os.DirFS("."),
	}
	for _, opt := range opts {
		opt(gm)
	}
	return gm
}

// GooseMigrator is a [pgtestdb.Migrator] that uses goose to perform migrations.
//
// Because Hash() requires calculating a unique hash based on the contents of
// the migrations, database, this implementation only supports reading migration
// files from disk or an embedded filesystem, and disables the global golang
// function migration registry.
//
// GooseMigrator does not allow specifying ExcludeNames or ExcludeVersions
// and will configure goose to run all the migrations ending in `*.sql` within
// the given filesystem and directory.
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
	ctx context.Context,
	db *sql.DB,
	_ pgtestdb.Config,
) error {
	store, err := database.NewStore(database.DialectPostgres, gm.TableName)
	if err != nil {
		return err
	}
	providerOptions := []goose.ProviderOption{
		goose.WithStore(store),
		goose.WithDisableGlobalRegistry(true),
	}
	migrationsDir, err := fs.Sub(gm.FS, gm.MigrationsDir)
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider("", db, migrationsDir, providerOptions...)
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}
