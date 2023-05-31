package goosemigrator

import (
	"context"
	"database/sql"
	"io/fs"
	"sync"

	"github.com/pressly/goose/v3"

	"github.com/peterldowns/testdb"
	"github.com/peterldowns/testdb/migrators/common"
)

// The mutex here makes this Migrator concurrency-safe. Goose uses Postgres
// advisory locks as off /v4, which is great, but doesn't prevent issues where
// multiple different tests running in parallel may attempt to run migrations at
// the same time.
var gooseLock sync.Mutex //nolint:gochecknoglobals

// Goose doesn't provide a constant for the default value.
var DefaultTableName = goose.TableName() //nolint:gochecknoglobals

type Option func(*GooseFSMigrator)

// default goose_db_version
// -table
// https://github.com/pressly/goose#usage
func WithTableName(tableName string) Option {
	return func(gm *GooseFSMigrator) {
		gm.TableName = tableName
	}
}

func WithFS(dir fs.FS) Option {
	return func(gm *GooseFSMigrator) {
		gm.FS = dir
	}
}

func New(migrationsDir string, opts ...Option) *GooseFSMigrator {
	gm := &GooseFSMigrator{
		MigrationsDir: migrationsDir,
		TableName:     DefaultTableName,
	}
	for _, opt := range opts {
		opt(gm)
	}
	return gm
}

type GooseFSMigrator struct {
	TableName     string
	MigrationsDir string
	FS            fs.FS
}

func (gm *GooseFSMigrator) Hash() (string, error) {
	hash := common.NewRecursiveHash(
		common.Field("TableName", gm.TableName),
	)
	if err := hash.AddDirs(gm.FS, "*.sql", gm.MigrationsDir); err != nil {
		return "", err
	}
	return hash.String(), nil
}

// Migrate runs migrate.Up() to migrate the template database.
func (gm *GooseFSMigrator) Migrate(
	_ context.Context,
	db *sql.DB,
	_ testdb.Config,
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
func (*GooseFSMigrator) Prepare(
	_ context.Context,
	_ *sql.DB,
	_ testdb.Config,
) error {
	return nil
}

// Verify is a no-op method.
func (*GooseFSMigrator) Verify(
	_ context.Context,
	_ *sql.DB,
	_ testdb.Config,
) error {
	return nil
}
