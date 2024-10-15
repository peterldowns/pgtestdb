# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

<!-- 
### Added
### Fixed
### Changed
### Deprecated
### Removed
### Security
-->

## [v0.1.1] - 2024-10-15

### Bugfix: GooseMigrator.Migrate() "dialect must be empty when using a custom store implementation"

This bug was first detected immediatley after pushing v0.1.0; CI passed locally
but [failed on main](https://github.com/peterldowns/pgtestdb/actions/runs/11353832019/job/31579725492) with this error:

```
 ?   	github.com/peterldowns/pgtestdb/internal/withdb	[no test files]
ok  	github.com/peterldowns/pgtestdb	2.544s
ok  	github.com/peterldowns/pgtestdb/internal/multierr	0.003s
ok  	github.com/peterldowns/pgtestdb/internal/once	0.003s
ok  	github.com/peterldowns/pgtestdb/internal/sessionlock	0.289s
ok  	github.com/peterldowns/pgtestdb/migrators/atlasmigrator	0.566s
ok  	github.com/peterldowns/pgtestdb/migrators/bunmigrator	0.306s
ok  	github.com/peterldowns/pgtestdb/migrators/common	0.010s
ok  	github.com/peterldowns/pgtestdb/migrators/dbmatemigrator	0.475s
ok  	github.com/peterldowns/pgtestdb/migrators/golangmigrator	0.286s
?   	github.com/peterldowns/pgtestdb/migrators/pgmigrator/migrations	[no test files]
--- FAIL: TestGooseMigratorFromDisk (0.07s)
    goose_test.go:21: failed to migrator.Migrate template testdb_tpl_b975d4cf2b5b60296612fea0fe385858: dialect must be empty when using a custom store implementation
--- FAIL: TestGooseMigratorFromFS (0.08s)
    goose_test.go:66: failed to migrator.Migrate template testdb_tpl_f65054fab233eabdfb95c95f7bf2e12a: dialect must be empty when using a custom store implementation
FAIL
FAIL	github.com/peterldowns/pgtestdb/migrators/goosemigrator	0.084s
ok  	github.com/peterldowns/pgtestdb/migrators/pgmigrator	0.317s
ok  	github.com/peterldowns/pgtestdb/migrators/sqlmigrator	0.217s
ok  	github.com/peterldowns/pgtestdb/migrators/ternmigrator	0.186s
FAIL
```

I introduced the bug in the `Migrate()` method, but did not detect it locally because
the migrator's `Hash()` resolved to a database template that I had already created
in my local postgres server. So basically:

- Work on the code, run tests, everything works and a database template is created
  for me locally.
- Work on the code some more, introduce the breaking problem, don't notice because
  when the tests run they re-use the existing database template and don't actually
  call `Migrate()`
- Release and push the code, resulting in CI failures on main because the database
  template doesn't exist so the tests have to run the broken `Migrate()` method.

Next time, I can avoid this type of problem by either:

- Using Github branches + PRs to merge in changes; CI is required to pass there before
  merging.
- Remembering to drop and recreate the test database server when working on changes that
  modify the `Migrate()` method.

Sloppy; my apologies.

## [v0.1.0] - 2024-10-15

### *Breaking*: require go1.21.0+, drop support for go1.18, go1.19, go1.20

[jackc/pgx/v5@latest](https://github.com/jackc/pgx/) uses some modern golang features
like the `slices` package, therefore requiring consumers to use go1.21 or higher.

Additionally, some of the migrators now require 1.21+ as well.

The [official golang release policy](https://go.dev/doc/devel/release#policy) is:

> Each major Go release is supported until there are two newer major releases.

Since go1.23 and go1.22 have been released, go1.21 isn't really supported any
more, but it seems like a reasonable target and is about a year old. Ratcheting
up from go1.18+ to go1.21+ seems fine to me and will allow some small quality of
life improvements and make contributing easier.

### *Breaking*: remove `Prepare()` and `Verify()` from `pgtestdb.Migrator`

```go
// Now:
type Migrator interface {
    Hash() (string, error)
    Migrate(context.Context, *sql.DB, Config) error
}
// Before:
type Migrator interface {
    Hash() (string, error)
    Prepare(context.Context, *sql.DB, Config) error
    Migrate(context.Context, *sql.DB, Config) error
    Verify(context.Context, *sql.DB, Config) error
}
```

The `Prepare()` method was removed because it was rarely used and none of the
migrators implemented it, leading to confusion about when/how to implement it.

- https://github.com/peterldowns/pgtestdb/issues/19
- https://github.com/peterldowns/pgtestdb/pull/6


If you were implementing `Prepare()` in your custom migrator, you can achieve the same
effect by moving that logic to the beginning of your `Migrate()` function:

```go
type MyCustomMigrator struct {
    // ...
}

func (m *MyCustomMigrator) Migrate(ctx context.Context, db *sql.DB, cfg Config) error {
    // Code that was previously run in the `Prepare()` method
    if err := m.DoPreparations(ctx, db, cfg); err != nil {
        return err
    }
    // Migration logic
    if err := m.ApplyMigrationsLikeBefore(ctx, db, cfg); err != nil {
        return err
    }
    // If desired, any post-migrations customization, like inserting fixture data
    if err := m.DoPostMigrationLogic(ctx, db, cfg); err != nil {
        return err
    }
    return nil
}
```

The `Verify()` method was only implemented by the `pgmigrator.Migrator`, and
almost never detected any problems. `Verify()` was included in the interface
because way back in the day, when I was first working on this library, I was
somehow able to corrupt the template database prepared by the migrators.  When
tests ran, they would create an instance from the existing template, but the
template was missing all the tables. Or something like that. At this point, I
believe those consistency issues have been addressed.  I have not seen a
verification error since the public release of pgtestdb. The method is somewhat
confusing, and I believe it's safe to remove it.

If you were relying on logic in the `Verify` method, you can move that logic to
your `NewDB(t *testing.T)` helper function:

```go
func NewDB(t *testing.T) *sql.DB {
  t.Helper()
  conf := pgtestdb.Config{
    DriverName: "pgx",
    User:       "postgres",
    Password:   "password",
    Host:       "localhost",
    Port:       "5433",
    Options:    "sslmode=disable",
  }

  var migrator pgtestdb.Migrator = &MyCustomMigrator{/* ... */}
  instanceConf := pgtestdb.Custom(t, conf, migrator)
  db, err := instanceConf.Connect()
  if err != nil {
    t.Fatalf("failed to connect to instance: %s", err)
  }

  // Run the Verify logic that was previously in the `Verify()` method of `MyCustomMigrator`
  if err := DoVerify(context.Background(), db, instanceConf); err != nil {
    t.Fatalf("failed to verify instance: %s", err)
  }
  return db
}
```

### Non-breaking: update [goosemigrator](/migrators/goosemigrator/) to use a [goose.Provider](https://pressly.github.io/goose/documentation/provider/)

Goose v3 [added a new goose.Provider](https://pressly.github.io/goose/blog/2023/goose-provider/) type to allow
users to run migrations without referencing a single package-level global variable. This is mostly an implementation
detail but could result in warnings from the golang race detector in certain cases.

Previously, the goosemigrator used a RW-lock to guard access to this shared
global and prevent race-detector warnings. Now that the `goose.Provider` is
available, the `goosemigrator` has been updated to use that instead.

The `goosemigrator.GooseMigrator` interface is unchanged, and should behave the
same as previously, but is now implemented differently. If you notice any
problems please report them and I'll do my best to fix them.

### Non-breaking: other tweaks

- The docker-compose and github actions files now use postgres:15 for testing
  pgtestdb. Previously we used postgis:15 in order to test that the default role
  was not allowed to enable superuser extensions, but I was able to update the
  tests to use the `pg_stat_statements` extension instead of `postgis` and confirm
  the same behavior. This makes developing against pgtestdb slightly easier and
  shouldn't impact correctness.

- In VSCode, if you open up any of the golang stdlib code files, the `gopls` extension
  [runs its analyses](https://github.com/golang/tools/blob/master/gopls/doc/analyzers.md) on them.
  This is annoying because it fills the VSCode "problems" view with lint errors that we cannot fix,
  because they're in the standard library, not our project's code. The only error I noticed was
  due to `unusedparams`, which we already lint for in our code with `golangci-lint`, so I updated
  the shared VSCode settings to turn off this gopls analysis.

 - Running `just tidy` updates all the `go.mod` files for the main library and
   the migrators, but these should all be backwards-compatible changes. 
