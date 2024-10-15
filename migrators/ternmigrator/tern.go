package ternmigrator

import (
	"context"
	"database/sql"
	"io/fs"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/common"
)

var _ pgtestdb.Migrator = (*TernMigrator)(nil)

// DefaultTableName is the default name for tern's migration table. This is
// the same as the default value in the tern command line tool.
const DefaultTableName = "public.schema_version"

// Option provides a way to configure the TernMigrator struct and its behavior.
//
// See:
//   - [WithTableName]
//   - [WithFS]
type Option func(*TernMigrator)

// WithFS specifies a `fs.FS` from which to read the migration files.
//
// Default: `<nil>` (reads from the real filesystem)
func WithFS(dir fs.FS) Option {
	return func(tm *TernMigrator) { tm.FS = dir }
}

// WithTableName specifies the name of the table in which tern will store its
// migration records.
//
// Default: `"public.schema_version"`
func WithTableName(tableName string) Option {
	return func(tm *TernMigrator) { tm.TableName = tableName }
}

// New returns a [TernMigrator]
//
// You can configure the behavior of the TernMigrator by passing Options:
//   - [WithFS] allows you to use an embedded filesystem.
//   - [WithTableName] is the name of the table in which tern will store its
func New(migrationsDir string, opts ...Option) *TernMigrator {
	tm := &TernMigrator{
		MigrationsDir: migrationsDir,
		TableName:     DefaultTableName,
	}
	for _, opt := range opts {
		opt(tm)
	}
	return tm
}

// TernMigrator is a pgtestdb.Migrator that uses tern to perform migrations.
type TernMigrator struct {
	TableName     string
	MigrationsDir string
	FS            fs.FS
}

// Hash returns a hash of the migrations.
func (tm *TernMigrator) Hash() (string, error) {
	hash := common.NewRecursiveHash(common.Field("TableName", tm.TableName))
	err := hash.AddDirs(tm.FS, "*.sql", tm.MigrationsDir)
	if err != nil {
		return "", err
	}
	return hash.String(), nil
}

func (tm *TernMigrator) fsys() (fs.FS, error) {
	if tm.FS == nil {
		return os.DirFS(tm.MigrationsDir), nil
	}
	return fs.Sub(tm.FS, tm.MigrationsDir)
}

// Migrate migrates the template database.
func (tm *TernMigrator) Migrate(ctx context.Context, _ *sql.DB, config pgtestdb.Config) (errOut error) {
	conn, err := pgx.Connect(ctx, config.URL())
	if err != nil {
		return err
	}
	defer func() {
		closeErr := conn.Close(ctx)
		if errOut == nil {
			errOut = closeErr
		}
	}()
	fsys, err := tm.fsys()
	if err != nil {
		return err
	}
	mig, err := migrate.NewMigrator(ctx, conn, tm.TableName)
	if err != nil {
		return err
	}
	err = mig.LoadMigrations(fsys)
	if err != nil {
		return err
	}
	return mig.Migrate(ctx)
}
