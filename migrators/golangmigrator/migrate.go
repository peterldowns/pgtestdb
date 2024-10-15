package golangmigrator

import (
	"context"
	"database/sql"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // pgx driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // "file://"" source driver

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/common"
)

// Option provides a way to configure the GolangMigrator struct and its behavior.
//
// golang-migrate documentation: https://github.com/golang-migrate/migrate
//
// See:
//   - [WithFS]
type Option func(*GolangMigrator)

// WithFS specifies a `fs.FS` from which to read the migration files.
//
// Default: `<nil>` (reads from the real filesystem)
func WithFS(dir fs.FS) Option {
	return func(gm *GolangMigrator) {
		gm.FS = dir
	}
}

// New returns a [GolangMigrator], which is a pgtestdb.Migrator that
// uses golang-migrate to perform migrations.
//
// `migrationsDir` is the path to the directory containing migration files.
//
// You can configure the behavior of dbmate by passing Options:
//   - [WithFS] allows you to use an embedded filesystem.
func New(migrationsDir string, opts ...Option) *GolangMigrator {
	gm := &GolangMigrator{
		MigrationsDir: migrationsDir,
		FS:            nil,
	}
	for _, opt := range opts {
		opt(gm)
	}
	return gm
}

// GolangMigrator is a pgtestdb.Migrator that uses golang-migrate to perform migrations.
//
// Because Hash() requires calculating a unique hash based on the contents of
// the migrations, database, this implementation only supports reading migration
// files from disk or an embedded filesystem.
//
// GolangMigrator does not perform any Verify() or Prepare() logic.
type GolangMigrator struct {
	// Where the migrations come from
	MigrationsDir string
	FS            fs.FS
}

func (gm *GolangMigrator) Hash() (string, error) {
	return common.HashDirs(gm.FS, "*.sql", gm.MigrationsDir)
}

// Migrate runs migrate.Up() to migrate the template database.
func (gm *GolangMigrator) Migrate(
	_ context.Context,
	_ *sql.DB,
	templateConfig pgtestdb.Config,
) error {
	var err error
	var m *migrate.Migrate
	if gm.FS == nil {
		m, err = migrate.New("file://"+gm.MigrationsDir, templateConfig.URL())
	} else {
		var d source.Driver
		if d, err = iofs.New(gm.FS, gm.MigrationsDir); err == nil {
			m, err = migrate.NewWithSourceInstance("iofs", d, templateConfig.URL())
		}
	}
	if err != nil {
		return err
	}
	defer m.Close()
	return m.Up()
}
