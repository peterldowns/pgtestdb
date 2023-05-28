package goosemigrator

import (
	"context"
	"database/sql"
	"io/fs"

	"github.com/pressly/goose/v3"

	"github.com/peterldowns/testdb"
	"github.com/peterldowns/testdb/migrators/common"
)

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
	return common.HashDirs(gm.FS, "*.sql", gm.MigrationsDir)
}

// Migrate runs migrate.Up() to migrate the template database.
func (gm *GooseFSMigrator) Migrate(
	_ context.Context,
	db *sql.DB,
	_ testdb.Config,
) error {
	goose.SetBaseFS(gm.FS)
	if gm.TableName != "" {
		goose.SetTableName(gm.TableName)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, gm.MigrationsDir)
}

// Prepare is a no-op method.
func (gm *GooseFSMigrator) Prepare(
	_ context.Context,
	_ *sql.DB,
	_ testdb.Config,
) error {
	return nil
}

// Verify is a no-op method.
func (gm *GooseFSMigrator) Verify(
	_ context.Context,
	_ *sql.DB,
	_ testdb.Config,
) error {
	return nil
}
