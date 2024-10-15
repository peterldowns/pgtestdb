package bunmigrator

import (
	"context"
	"database/sql"
	"io/fs"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/migrate"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/common"
)

var _ pgtestdb.Migrator = (*BunMigrator)(nil)

// Option provides a way to configure the BunMigrator struct and its behaviour.
//
// bun migration documentation: https://bun.uptrace.dev/guide/migrations.html
//
// See:
//   - [WithFS]
//   - [WithBunDBOpts]
//   - [WithMigratorOpts]
//   - [WithMigrationOpts]
type Option func(*BunMigrator)

// WithFS specifies a `fs.FS` from which to read the migration files.
//
// Default: `<nil>` (reads from the real filesystem)
func WithFS(dir fs.FS) Option {
	return func(bm *BunMigrator) {
		bm.FS = dir
	}
}

// WithBunDBOpts passes options to the bun.DB struct.
func WithBunDBOpts(opts ...bun.DBOption) Option {
	return func(bm *BunMigrator) {
		bm.BunDBOpts = opts
	}
}

// WithMigratorOpts passes options to the migrate.Migrator.
func WithMigratorOpts(opts ...migrate.MigratorOption) Option {
	return func(bm *BunMigrator) {
		bm.MigratorOpts = opts
	}
}

// WithMigrationOpts passes options to the migrate.Migrator.Migrate() func.
func WithMigrationOpts(opts ...migrate.MigrationOption) Option {
	return func(bm *BunMigrator) {
		bm.MigrationOpts = opts
	}
}

// New returns a [BunMigrator], which is a pgtestdb.Migrator that
// uses bun to perform migrations.
//
// `migrationsDir` is the path to the directory containing migration files.
//
// You can configure the behaviour of bun by passing Options:
//   - [WithFS] allows you to use an embedded filesystem.
//   - [WithBunDBOpts] allows you to pass options to the underlying bun.DB struct.
//   - [WithMigrationOpts] allows you to pass options to the Migrate() function.
//   - [WithMigratorOpts] allows you to pass options to the Migrator struct.
func New(migrationsDir string, opts ...Option) *BunMigrator {
	bm := &BunMigrator{
		MigrationsDir: migrationsDir,
		FS:            nil,
	}
	for _, opt := range opts {
		opt(bm)
	}
	return bm
}

// BunMigrator is a pgtestdb.Migrator that uses bun to perform migrations.
//
// Because Hash() requires calculating a unique hash based on the contents of
// the migrations, this implementation only supports reading migration files
// from disk or an embedded filesystem.
//
// BunMigrator does not perform any Verify() logic.
type BunMigrator struct {
	MigrationsDir string
	FS            fs.FS
	BunDBOpts     []bun.DBOption
	MigratorOpts  []migrate.MigratorOption
	MigrationOpts []migrate.MigrationOption
}

func (bm *BunMigrator) Hash() (string, error) {
	return common.HashDirs(bm.FS, "*.sql", bm.MigrationsDir)
}

// Migrate migrates the template database.
func (bm *BunMigrator) Migrate(ctx context.Context, sqldb *sql.DB, _ pgtestdb.Config) error {
	var err error
	migrations := migrate.NewMigrations()
	if bm.FS == nil {
		err = migrations.Discover(os.DirFS(bm.MigrationsDir))
	} else {
		err = migrations.Discover(bm.FS)
	}
	if err != nil {
		return err
	}
	db := bun.NewDB(sqldb, pgdialect.New(), bm.BunDBOpts...)
	m := migrate.NewMigrator(db, migrations, bm.MigratorOpts...)
	// Initialize the bun migrator, creating the tables that keep track of which
	// migrations have been applied.
	err = m.Init(ctx)
	if err != nil {
		return err
	}
	// Apply the migrations.
	if _, err := m.Migrate(ctx, bm.MigrationOpts...); err != nil {
		return err
	}
	return nil
}
