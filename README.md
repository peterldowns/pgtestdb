# 🧪 pgtestdb

![Latest Version](https://badgers.space/badge/latest%20version/v0.1.1/blueviolet?corner_radius=m)
![Golang](https://badgers.space/badge/golang/1.21+/blue?corner_radius=m)

pgtestdb is a golang library that helps you write efficient database-backed tests.
It uses [template
databases](https://www.postgresql.org/docs/current/manage-ag-templatedbs.html)
to give each test a fully prepared and migrated Postgres database &mdash; 
no mocking, no cleanup, no hassle. Your migrations only run once and each test
only waits for ~20ms to get its own database. Comes with connectors for the most
popular migration frameworks, works with everything.

# Documentation

- [The Github README, https://github.com/peterldowns/pgtestdb](https://github.com/peterldowns/pgtestdb)
- [The go.dev docs, pkg.go.dev/github.com/peterldowns/pgtestdb](https://pkg.go.dev/github.com/peterldowns/pgtestdb)

The Github README is the primary source for documentation. The code itself is
supposed to be well-organized, and each function has a meaningful docstring, so
you should be able to explore it quite easily using an LSP plugin, reading the
code, or clicking through the go.dev docs.

## Changelog

As of October 2024, all breaking changes, and most non-breaking changes, are
documented in the [CHANGELOG](/CHANGELOG.md).


## How it works
Each time one of your tests asks for a fresh database by calling `pgtestdb.New`, pgtestdb will
check to see if a template database already exists. If not, it creates a new
database, runs your migrations on it, and then marks it as a template. Once the
template exists, it then creates a test-specific database from that template.

Creating a new database from a template is _very_ fast, on the order of 10s of
milliseconds.  And because pgtestdb uses advisory locks and hashes your
migrations to determine which template database to use, your migrations only
end up being run one time, regardless of how many tests or separate packages
you have. This is true even across test runs --- pgtestdb will only run your
migrations again if you change them in some way.

When a test succeeds, the database it used is automatically deleted.
When a test fails, the database it used is left alive, and the test logs will
include a connection string you can use to connect to it with `psql` and explore
what happened.

pgtestdb is concurrency-safe &mdash; because each of your tests gets its own
database, you can and should run your tests in parallel.

## Supported Migration Frameworks
pgtestdb works with any migration framework, and includes out-of-the-box adapters
for the most popular golang frameworks:

- [pgmigrator](migrators/pgmigrator/) for [peterldowns/pgmigrate](https://github.com/peterldowns/pgmigrate)
- [golangmigrator](migrators/golangmigrator/) for [golang-migrate/migrate](https://github.com/golang-migrate/migrate)
- [goosemigrator](migrators/goosemigrator/) for [pressly/goose](https://github.com/pressly/goose)
- [dbmatemigrator](migrators/dbmatemigrator/) for [amacneil/dbmate](https://github.com/amacneil/dbmate)
- [atlasmigrator](migrators/atlasmigrator/) for [ariga/atlas](https://github.com/ariga/atlas)
- [sqlmigrator](migrators/sqlmigrator/) for [rubenv/sql-migrate](https://github.com/rubenv/sql-migrate)
- [bunmigrator](migrators/bunmigrator/) for [uptrace/bun](https://github.com/uptrace/bun) (contributed by [@BrynBerkeley](https://github.com/BrynBerkeley))
- [ternmigrator](migrators/ternmigrator/) for [jackc/tern](https://github.com/jackc/tern) (contributed by [@WillAbides](https://github.com/WillAbides))

You can use pgtestdb with any migration tool: see the
[`pgtestdb.Migrator`](#pgtestdbmigrator) docs for more information on writing
your own adapter.

## Install

```shell
go get github.com/peterldowns/pgtestdb@latest
```

## Quickstart

### Example Test

Here's how you use `pgtestdb.New` in a test to get a database.

```go

// pgtestdb uses the `sql` interfaces to interact with Postgres, you just have to
// bring your own driver. Here we're using the PGX driver in stdlib mode, which
// registers a driver with the name "pgx".
import (
  // ...
  _ "github.com/jackc/pgx/v5/stdlib"
  // ...
)

func TestMyExample(t *testing.T) {
  // pgtestdb is concurrency safe, enjoy yourself, run a lot of tests at once
  t.Parallel()
  // You should connect as an admin user. Use a dedicated server explicitly
  // for tests, do NOT use your production database.
  conf := pgtestdb.Config{
    DriverName: "pgx",
    User:       "postgres",
    Password:   "password",
    Host:       "localhost",
    Port:       "5433",
    Options:    "sslmode=disable",
  }
  // You'll want to use a real migrator, this is just an example. See
  // the rest of the docs for more information.
  var migrator pgtestdb.Migrator = pgtestdb.NoopMigrator{}
  db := pgtestdb.New(t, conf, migrator)
  // If there is any sort of error, the test will have ended with t.Fatal().
  // No need to check errors! Go ahead and use the database.
  var message string
  err := db.QueryRow("select 'hello world'").Scan(&message)
  assert.Nil(t, err)
  assert.Equal(t, "hello world", message)
}
```

### Defining A Helper

It would be tedious to add that whole prelude to each of your tests. I recommend
that you define a test helper function that calls `pgtestdb.New` with the same
`pgtestdb.Config` and `pgtestdb.Migrator` each time. You should then use this helper
in your tests. Here is an example:

```go
// NewDB is a helper that returns an open connection to a unique and isolated
// test database, fully migrated and ready for you to query.
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
  // You'll want to use a real migrator, this is just an example. See the rest
  // of the docs for more information.
  var migrator pgtestdb.Migrator = pgtestdb.NoopMigrator{}
  return pgtestdb.New(t, conf, migrator)
}
```

Call this helper in each test. You'll either have a valid `*sql.DB` connection to
a test database, or the test will fail and stop executing.

```go
func TestAQuery(t *testing.T) {
  t.Parallel()
  db := NewDB(t) // this is the helper defined above

  var result string
  err := db.QueryRow("SELECT 'hello world'").Scan(&result)
  check.Nil(t, err)
  check.Equal(t, "hello world", result)
}
```

### Running The Postgres Server
pgtestdb requires you to provide a connection to a Postgres server. I **strongly
recommend** running a dedicated server just for tests that is RAM-backed
(instead of disk-backed) and tuned for performance by removing all data-sync
guarantees. This would be a bad idea in production, but in tests it works great.
Your tests will go ⚡️ *fast* ⚡️.

**Do Not** connect pgtestdb to the same server that
contains your production data. pgtestdb requires admin privileges to work, and
creates and deletes databases as part of its operation. You should always use a
dedicated server for your tests.

pgtestdb will connect to any postgres server as long as you can supply
the username, password, host, port, and database name -- the config
generates a postgres connection string of the form `postgres://user:password@host:port/dbname?options`.

Some common methods of running a Postgres server for pgtestdb:

- run Postgres inside Docker / with Docker Compose
- run Postgres natively, through a binary or package you install
- run Postgres on a remote server somewhere

There are some projects, like 
[ory/dockertest](https://github.com/ory/dockertest/blob/v3/examples/PostgreSQL.md)
or
[fergusstrange/embedded-postgres](https://github.com/fergusstrange/embedded-postgres),
that allow you to write a `TestMain(m *test.M)` method and spin up a postgres
server from your golang code. I **strongly recommend you do not use these
libraries**, they were generally written with a different testing model in
mind. Two major drawbacks:

* these methods generally create a postgres server for each package you're
  testing (in the `TestMain(m *testing.M)`) instead of sharing a single
  postgres server for all the packages that you're testing. This means that if
  you're testing N packages, your migrations will run N times, and your tests
  will be that much slower.
* these packages are always brittle in that they do not guarantee the server
  exits cleanly, resulting in leaked server processes (at best) or stateful
  failures where servers collide with each other and cannot start (at worst.)

Instead, I **strongly recommend using Docker Compose** to run a single postgres
server.  Developers are often very familiar with Docker, it's generally easy to
use in CI, and it makes it easy to use a tmpfs/ramdisk for the file system.

Here is an example `docker-compose.yml` file you can use to run a RAM-backed
Postgres server inside of a docker container. For more performance tuning
options, see the FAQ below.

```yaml
version: "3.6"
services:
  pgtestdb:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: password
    restart: unless-stopped
    volumes:
      # Uses a tmpfs volume to make tests extremely fast. The data in test
      # databases is not persisted across restarts, nor does it need to be.
      - type: tmpfs
        target: /var/lib/postgresql/data/
    command:
      - "postgres"
      - "-c" # turn off fsync for speed
      - "fsync=off"
      - "-c" # log everything for debugging
      - "log_statement=all"
    ports:
      # Entirely up to you what port you want to use while testing.
      - "5433:5432"
```

If you do not have a server running, or pgtestdb cannot connect to your server,
you will see a failure message like this one:

```
--- FAIL: TestAQuery (0.00s)
    /Users/pd/code/example/example_test.go:170: failed to provision test database template: failed to connect to `host=localhost user=postgres database=`: dial error (dial tcp [fe80::1%lo0]:5433: connect: connection refused)
```

### Choosing A Driver

As part of creating and migrating the test databases, pgtestdb will connect to
the server via the `sql.DB` database interface. In order to do so, you will need
to choose, register, and configure your SQL driver. pgtestdb will work with
[pgx](https://github.com/jackc/pgx/) or [lib/pq](https://github.com/lib/pq) or
any other [database/sql driver](https://pkg.go.dev/database/sql/driver). I
recommend using the pgx driver unless you have a good reason to remain on
lib/pq. 

As with any sql driver,  make sure to import the driver so that it registers
itself. Then, pass its name in the `pgtestdb.Config`:

```go
import (
  // Makes both drivers available as an example.
  _ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver
  _ "github.com/lib/pq"              // registers the "postgres" driver
)

func TestWithPgxStdlibDriver(t *testing.T) {
  t.Parallel()
  pgxConf := pgtestdb.Config{
    DriverName: "pgx", // uses the pgx/stdlib driver
    User:       "postgres",
    Password:   "password",
    Host:       "localhost",
    Port:       "5433",
    Options:    "sslmode=disable",
  }
  migrator := pgtestdb.NoopMigrator{}
  db := pgtestdb.New(t, pgxConf, migrator)

  var message string
  err := db.QueryRow("select 'hello world'").Scan(&message)
  assert.Nil(t, err)
  assert.Equal(t, "hello world", message)
}

func TestWithLibPqDriver(t *testing.T) {
  t.Parallel()
  pqConf := pgtestdb.Config{
    DriverName: "postgres", // uses the lib/pq driver
    User:       "postgres",
    Password:   "password",
    Host:       "localhost",
    Port:       "5433",
    Options:    "sslmode=disable",
  }
  migrator := pgtestdb.NoopMigrator{}
  db := pgtestdb.New(t, pqConf, migrator)

  var message string
  err := db.QueryRow("select 'hello world'").Scan(&message)
  assert.Nil(t, err)
  assert.Equal(t, "hello world", message)
}

```

## API
### `pgtestdb.New`

```go
func New(t testing.TB, conf Config, migrator Migrator) *sql.DB
```

`New` creates and connects to a new test database, and ensures that all migrations are run. If any part of this fails, the test is marked as a failure using `t.Fail()`

[`testing.TB`](https://pkg.go.dev/testing#TB) is the common testing interface implemented by `*testing.T`, `*testing.B`, and `*testing.F`, so you can use pgtestdb to get a database for tests, benchmarks, and fuzzes.

How does it work? Each time it's called, it:

- Connects to a running Postgres server using the provided config.
- Ensures that there is a role `USER=pgtdbuser PASSWORD=pgtdbpass`.
- Calls `Hash()` on the provided migrator to determine the name of the template
 database.
- If the template database does not exist:
  - Creates a new, empty, database.
  - Gives `pgtdbuser` ownership of this database and all of its contents
    (schemas, tables, sequences).
  - Calls `Migrate()` on the provided migrator to actually migrate the database schema.
  - Marks the database as a template
- Creates a new database instance from the template

It will use both golang-level locks and Postgres-level [advisory
locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS)
to synchronize, meaning that your migrations will only run one time no matter
how many tests or packages are being tested in parallel.

Once it creates your brand new fresh test database, pgtestdb will `t.Log()` the
connection string to the database instance.

If your test fails, the logs will show that connection string, and you can connect to it
with any of your usual tools (`psql`) to help you debug by looking at the data left there
at the end of the test.

If your test passes, a hook registered with `t.Cleanup()` will remove the database
instance that was used by the test.


### `pgtestdb.Custom` 

```go
func Custom(t testing.TB, conf Config, migrator Migrator) *Config
```

`Custom` is like `New` but after creating the new database instance, it closes
its connection and returns the configuration details of that database so that
you can connect to it explicitly, potentially via a different SQL interface.

You can get the connection URL for the database by calling `.URL()` on the
config (see below.)

### `pgtestdb.Config`

```go
// Config contains the details needed to connect to a postgres server/database.
type Config struct {
    DriverName string // the name of a driver to use when calling sql.Open() to connect to a database, "pgx" (pgx) or "postgres" (lib/pq)
    Host       string // the host of the database, "localhost"
    Port       string // the port of the database, "5433"
    // User is the role that pgtestdb connects to the server with in order to
    // create and manage databases that are used by the tests. Your user should
    // have the SUPERUSER, CREATEDB, and CREATEROLE capabilities.
    User       string // the user to connect as, "postgres"
    Password   string // the password to connect with, "password"
    Database   string // the database to connect to, "postgres"
    Options    string // URL-formatted additional options to pass in the connection string, "sslmode=disable&something=value"
    // TestRole is the role used to create and connect to the template database
    // and each test database. If not provided, defaults to [DefaultRole].  The
    // capabilities of this role should match the capabilities of the role that
    // your application uses to connect to its database and run migrations.
    TestRole *Role
    // If true, ForceTerminateConnections will force-disconnect any remaining
    // database connections prior to dropping the test database. This may be
    // necessary if your code leaks database connections, intentionally or
    // unintentionally. By default, if you leak a connection to a test database,
    // pgtestdb will be unable to drop the database, and the test will be failed
    // with a warning.
    ForceTerminateConnections bool
}

// URL returns a postgres connection string in the format
// "postgres://user:password@host:port/database?options=..."
func (c Config) URL() string

// Connect calls `sql.Open()“ and connects to the database.
func (c Config) Connect() (*sql.DB, error)
```

The `Config` struct contains the details needed to connect to a Postgres server.
Make sure to connect with a user that has the necessary capabilities to create
new databases and roles. Most likely you want to connect as the default
`postgres` user, since you'll be connecting to a dedicated testing-only Postgres
server as described earlier.

### `pgtestdb.Role`
A dedicated Postgres role (user) is used to create the template database and each test database. pgtestdb will create this role for you with sane defaults, but you can control the username, password, and capabilities of this role if desired.

```go
const (
    // DefaultRoleUsername is the default name for the role that is created and
    // used to create and connect to each test database.
    DefaultRoleUsername = "pgtdbuser"
    // DefaultRolePassword is the default password for the role that is created and
    // used to create and connect to each test database.
    DefaultRolePassword = "pgtdbpass"
    // DefaultRoleCapabilities is the default set of capabilities for the role
    // that is created and used to create and conect to each test database.
    // This is locked down by default, and will not allow the creation of
    // extensions.
    DefaultRoleCapabilities = "NOSUPERUSER NOCREATEDB NOCREATEROLE"
)

// DefaultRole returns the default Role used to create and connect to the
// template database and each test database.  It is a function, not a struct, to
// prevent accidental overriding.
func DefaultRole() Role {
    return Role{
        Username:     DefaultRoleUsername,
        Password:     DefaultRolePassword,
        Capabilities: DefaultRoleCapabilities,
    }
}

// Role contains the details of a postgres role (user) that will be used
// when creating and connecting to the template and test databases.
type Role struct {
    // The username for the role, defaults to [DefaultRoleUsername].
    Username string
    // The password for the role, defaults to [DefaultRolePassword].
    Password string
    // The capabilities that will be granted to the role, defaults to
    // [DefaultRoleCapabilities].
    Capabilities string
}
```  

Because this role is used to connect to each template and each test database
and run the migrations, its capabilities should match those of your production
application. For instance, if in production your application connects as a
superuser, you will want to pass a custom `Role` whthat includes the
`SUPERUSER` capability so that your migrations will run the same in both environments.

This is a common case for many applications that install or activate extensions
like [Postgis](https://postgis.net/), which require activation via a superuser.

### `pgtestdb.Migrator`

The `Migrator` interface contains all of the logic needed to prepare a template
database that can be cloned for each of your tests. pgtestdb requires you to
supply a `Migrator` to work. There are already migrators for the most popular migration frameworks, you can use these right away:

- [pgmigrator](migrators/pgmigrator/) for [peterldowns/pgmigrate](https://github.com/peterldowns/pgmigrate)
- [golangmigrator](migrators/golangmigrator/) for [golang-migrate/migrate](https://github.com/golang-migrate/migrate)
- [goosemigrator](migrators/goosemigrator/) for [pressly/goose](https://github.com/pressly/goose)
- [dbmatemigrator](migrators/dbmatemigrator/) for [amacneil/dbmate](https://github.com/amacneil/dbmate)
- [atlasmigrator](migrators/atlasmigrator/) for [ariga/atlas](https://github.com/ariga/atlas)
- [sqlmigrator](migrators/sqlmigrator/) for [rubenv/sql-migrate](https://github.com/rubenv/sql-migrate)
- [bunmigrator](migrators/bunmigrator/) for [uptrace/bun](https://github.com/uptrace/bun) (contributed by [@BrynBerkeley](https://github.com/BrynBerkeley))
- [ternmigrator](migrators/ternmigrator/) for [jackc/tern](https://github.com/jackc/tern) (contributed by [@WillAbides](https://github.com/WillAbides))

You can also write your own, and/or embed the existing migrators into your own
to run custom logic before/after running migrations.

```go
// A Migrator is necessary to provision the database that will be used as as template
// for each test.
type Migrator interface {
    // Hash should return a unique identifier derived from the state of the database
    // after it has been fully migrated. For instance, it may return a hash of all
    // of the migration names and contents.
    //
    // pgtestdb will use the returned Hash to identify a template database. If a
    // Migrator returns a Hash that has already been used to create a template
    // database, it is assumed that the database need not be recreated since it
    // would result in the same schema and data.
    Hash() (string, error)
    // Migrate is a function that actually performs the schema and data
    // migrations to provision a template database. The connection given to this
    // function is to an entirely new, empty, database. Migrate will be called
    // only once, when the template database is being created.
    Migrate(context.Context, *sql.DB, Config) error
}
```

If you're writing your own `Migrator`, I recommend you use the existing ones
as examples. Most migrators need to do some kind of file/directory hashing in
order to implement `Hash()` &mdash; you may want to use
[the helpers in the `common` subpackage](migrators/common).

For example and testing purposes, there is a no-op migrator that does
nothing at all.

```go
// NoopMigrator fulfills the Migrator interface but does absolutely nothing.
// You can use this to get empty databases in your tests, or if you are trying
// out pgtestdb and aren't sure which migrator to use yet.
//
// For more documentation on migrators, see
// https://github.com/peterldowns/pgtestdb#pgtestdbmigrator
type NoopMigrator struct{}
```

If you'd like to run custom code before or after running migrations, I recommend
writing a custom `Migrator` that embeds an existing `Migrator`. For details, see
[this example](#TODO).

# FAQ

## Is this real?
Yes, this is extremely real, and works as promised. Try it out and see!

## Has anyone ever done this before?
Some prior art on the concept of template databases and ramdisks for testing
against Postgres in general:

- https://github.com/allaboutapps/integresql
- http://eradman.com/ephemeralpg/

As far as I know, no one has made it this easy to do it though.

## Rough perf numbers?

If you're using a RAM-backed server and have about a thousand migrations, think
~500ms to prepare the template database one time, ~10ms to clone a template.

The time to prepare the template depends on your migration strategy and your
database schema.

The time to get a new clone seems pretty constant even for databases with large
schemas.

Everything depends on the speed of your server.

For an example benchmark, check out [hallabro/lightning-fast-database-tests](https://github.com/hallabro/lightning-fast-database-tests), which contains code + slides from a Gophercon 2024 lightning talk given by [Robin Hallabro-Kokko](https://github.com/hallabro).

## How do I make it go faster?
A ramdisk and turning off fsync is just the start &mdash; if you care about
performance, you should make sure to tune all the other options that Postgres
makes available.

The [official wiki has lots of links](https://wiki.postgresql.org/wiki/Performance_Optimization).

For ramdisk/CI in particular, you may get some ideas from [this blog post](https://www.manniwood.com/postgresql_94_in_ram/).

[Integresql](https://github.com/allaboutapps/integresql#run-using-docker-preferred) sets the following options:

```
-c 'shared_buffers=128MB'
-c 'fsync=off'
-c 'synchronous_commit=off'
-c 'full_page_writes=off'
-c 'max_connections=100'
-c 'client_min_messages=warning'
```

## Why are my tests failing because they can't connect to Postgres?

First, make sure the server is running and you can connect to it. But assuming
you're seeing some kind of failure while running a larger test suite, the most likely
problem is that you're exceeding the maximum number of connections your Postgres server is
configured to accept. This may show up as a few different types of error messages:

```
--- FAIL: TestParallel2/subtest_34 (0.01s)
    testdb_test.go:134: failed to create instance: failed to create instance from template testdb_tpl_ed8ae75db1176559951eadb85d6be0db: failed to connect to `host=localhost user=postgres database=`: server error (FATAL: sorry, too many clients already (SQLSTATE 53300))
```

or

```
--- FAIL: TestParallel2/subtest_25 (0.04s)
    testdb_test.go:134: could not create pgtestdb user: sessionlock(testdb-user) failed to open conn: failed to connect to `host=localhost user=postgres database=`: dial error (dial tcp 127.0.0.1:5433: connect: connection refused
  ```

or

```
--- FAIL: TestNew (0.07s)
    testdb_test.go:42: failed to migrator.Prepare template testdb_tpl_ed8ae75db1176559951eadb85d6be0db: sessionlock(test-sql-migrator) failed to open conn: failed to connect to `host=localhost user=pgtdbuser database=testdb_tpl_ed8ae75db1176559951eadb85d6be0db`: server error (FATAL: remaining connection slots are reserved for non-replication superuser connections (SQLSTATE 53300))
```

The fundamental way to fix this error is to make sure that you do not attempt
more simultaneous connections to the server than it is ready to accept. Here are
the best ways to do that, from easiest to most complicated:

* Just don't run so many tests at once. If you're developing locally, and have a fast CI, you may not need to run the full test suite at once.
* Set a higher value for your server's [`max_connections` parameter](https://www.postgresql.org/docs/current/runtime-config-connection.html), which defaults to 100. If you look at the [`docker-compose.yaml`](./docker-compose.yml) in this repo you'll see we set it to 1000. Assuming you're using a dockerized and tmpfs-backed test server on a beefy machine, you should be able to safely crank this really high.
* Run fewer tests at the same time. You'll want to read the [go docs for the `-parallel` test flag and the `-p` build flag](https://pkg.go.dev/cmd/go) very carefully. You can also find these docs by running `go help build` and `go help tesflags`. Basically, `-p N` means go will run tests for up to `N` packages at the same time, and `-parallel Y` means it will run up to `Y` tests in parallel within each package. Both `N` and `Y` default to `GOMAXPROCS`, which is the number of logical CPUs in your machine. So by default, `go test ./...` can run up to `GOMAXPROCS * GOMAXPROCS` tests at the same time. You can try tuning `parallel Y` and `-p N` to set a hard limit of `N * Y` simultaneous tests, but that doesn't mean that you're using at most `N * Y` database connections --- depending on your test and application logic, you may actually use multiple connections at the same time. Sorry, it's confusing.
* Consider putting a connection pooler in front of your test database. This is maybe easiest to do in CI but it's possible locally with a containerized instance of pgbouncer, for example. Your tests will still contend for your database's resources, but it may turn "tests failing" into "tests getting slower".

It's worth noting that when you call `pgtestdb.New()` or `pgtestdb.Custom()`, the library will only use one active connection at any point in time. So if you have 100 tests running in parallel, pgtestdb will at most consume 100 simultaneous connections. Your own code in your tests may run multiple queries in parallel and consume more connections at once, though.

## How do I connect to the test databases through `pgx` / `pgxpool` instead of using the sql.DB interface?

You can use `pgtestdb.Custom` to connect via `pgx`, `pgxpool`, or any other
method of your choice. Here's an example of connecting via `pgx`.

```go
// pgtestdb uses the `sql` interfaces to interact with Postgres, you just have to
// bring your own driver. Here we're using the PGX driver in stdlib mode, which
// registers a driver with the name "pgx".
import (
    // ...
    // register the PGX stdlib driver so that pgtestdb
    // can create the test database.
    _ "github.com/jackc/pgx/v5/stdlib"
    // import pgx so that we can use it to connect to the database
    "github.com/jackc/pgx/v5"
    // ...
)

func TestCustom(t *testing.T) {
    ctx := context.Background()
        dbconf := pgtestdb.Config{
        DriverName: "pgx",
        User:       "postgres",
        Password:   "password",
        Host:       "localhost",
        Port:       "5433",
        Options:    "sslmode=disable",
    }
    m := defaultMigrator()
    config := pgtestdb.Custom(t, dbconf, m)
    check.NotEqual(t, dbconf, *config)

    var conn *pgx.Conn
    var err error
    conn, err = pgx.Connect(ctx, config.URL())
    assert.Nil(t, err)
    defer func() {
        err := conn.Close(ctx)
        assert.Nil(t, err)
    }()

    var message string
    err = conn.QueryRow(ctx, "select 'hello world'").Scan(&message)
    assert.Nil(t, err)
}
```

## How do I connect to a database over unix domain socket like `/run/postgresql`?

To connect to a database that is listening on a domain socket, for instance `/run/postgresql`, you
will need to pass the socket path as a `host={socket path}` options parameter.

The [Postgresql documentation](https://www.postgresql.org/docs/current/libpq-connect.html#:~:text=The%20host%20part%20is%20interpreted%20as%20described%20for%20the%20parameter%20host.) describes two ways of connecting to socket paths using a `postgres://...` connection string:

> So, to specify a non-standard Unix-domain socket directory, **either omit the host part of the URI and specify the host as a named parameter**, or percent-encode the path in the host part of the URI

but due to the way the golang `sql.Open` is implemented, percent-encoding the path in the `Config.Host` field won't work.

So, you need to set `host={socket path}` (with appropriate URL encoding) in the `Config.Options` field:

```go
conf := pgtestdb.Config{
  // The same approach works for both "postgres" (lib/pq) and "pgx" (pgx)
  DriverName: "pgx",
  User:       "postgres",
  Port:       "5432",
  Password:   "password",
  // %2F is a url-encoded '/' character
  Options:    "host=%2Frun%2Fpostgresql",
  Database:   "postgres",
}
```

## Does this mean I should stop writing unit tests and doing dependency injection?
No! Please keep writing unit tests and doing dependency injection and mocking
and all the other things that make your code well-organized and easily
testable.

This project exists because the database is probably _not_ one of the things
that you want to be mocking in your tests, and most modern applications have a
large amount of logic in Postgres that is hard to mock anyway.

## How does this play out in a real company?
[Pipe](https://pipe.com) had thousands of tests using a similar package
to get template-based databases for each test. The whole test suite ran in under
a few minutes on reasonably-priced CI machines, and individual packages/tests
ran fast enough on local development machines that developers were happy to add
new database-backed tests without worrying about the cost.

I believe that pgtestdb and a ram-backed Postgres server is fast enough to be
worth it. If you try it out and don't think so, please open an issue &mdash;
I'd be very interested to see if I can make it work for you, too.

## How can I contribute?
pgtestdb is a standard golang project, you'll need a working golang environment.
If you're of the nix persuasion, this repo comes with a flakes-compatible
development shell that you can enter with `nix develop` (flakes) or `nix-shell`
(standard).

If you use VSCode, the repo comes with suggested extensions and settings.

Testing and linting scripts are defined with
[Just](https://github.com/casey/just), see the [Justfile](./Justfile) to see how
to run those commands manually. There are also Github Actions that will lint and
test your PRs.

Contributions are more than welcome!
