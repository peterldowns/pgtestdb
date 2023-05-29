| :warning: WARNING |
|:--- |
| This is a Work In Progress |

# 🧪 testdb

testdb makes it cheap and easy to create ephemeral Postgres databases for your
golang tests. It uses template databases to give each test a fully prepared and
migrated Postgres database &mdash; no mocking, no cleanup, no hassle. Bring your
own migration framework, works with everything.

If you use
testdb, you and your team will write more useful tests that run faster and catch
more problems.

**Stop** worrying about the cost of adding tests that use your database.

**Stop** wasting time writing mocks and stubs for your database.

**Stop** arguing about the conjoined triangle of unit and integration tests.

**Start writing tests** that meaningfully exercise your application's actual
logic and behavior, using the full power of your database (including triggers,
views, extensions, etc.)


# How does it work?

testdb provides a function, `testdb.New(...) *sql.DB`, that each of your tests
can call to receive a connection to a brand-new fully-isolated,
totally-independent, blazing-fast database. Your tests can run in parallel
without any issue. If your test succeeds, the database is automatically removed
as part of the test cleanup. If your test fails, or while you're using a
debugger, the database is left alive and you can connect to it with `psql` or
other tools to inspect the state and data.

testdb works by running your migrations at most once, creating [a template
database](https://www.postgresql.org/docs/current/manage-ag-templatedbs.html).
Each time a test asks for a fresh database by calling `testdb.New(...)`, testdb
will check to see if a template database already exists. If not, it creates one
and runs your migrations.  Once the template exists, it is _very_ fast to
create a new database from that template.

testdb requires you to provide your own migration strategy, and to provide a 
method that returns a unique identifier for every set of migrations. This way,
if you don't modify your migrations, you can keep re-using the same template over
and over again. 

This alone isn't enough to make these tests fast enough to be impressive. The
next trick is that for testing purposes, you can run a Postgres server that
stores everything in RAM and turns off all disk syncing. This will make it go
:zap: *fast*.

Turning off disk syncing and storing data in RAM would be a terrible idea in
production, but this is one of the times that you should take advantage of the
fact that tests != production. It will work just fine until your computer
sleeps or shuts off. It will work perfectly in CI. And you'll be blown away by
how fast your tests run.

I recommend using the following settings, presented here in the form of a
`docker-compose.yml` file you can start using right now:

```yaml
version: "3.6"
services:
  testdb:
    image: postgres:13
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
      # turn off fsync for speed
      - "-c"
      - "fsync=off"
      # log everything for debugging
      - "-c"
      - "log_statement=all"
    ports:
      # Entirely up to you what port you want to use while testing.
      - "5433:5432"
```
# Install

```shell
go get github.com/peterldowns/testdb@latest
```

# Documentation

- [This page, https://github.com/peterldowns/testdb](https://github.com/peterldowns/testdb)
- [The go.dev docs, pkg.go.dev/github.com/peterldowns/testdb](https://pkg.go.dev/github.com/peterldowns/testdb)

This page is the primary source for documentation. The code itself is supposed
to be well-organized, and each function has a meaningful docstring, so you
should be able to explore it quite easily using an LSP plugin, reading the
code, or clicking through the go.dev docs.

## How to use it

You probably want to define your own helper function that calls
`testdb.New(...)` with the same `testdb.Config` and `testdb.Migrator` each time.

```go
package testhelpers

import (
  "database/sql"
  "testing"

  "github.com/peterldowns/testdb"
)

// NewDB is a helper that returns an open connection to a unique and isolated
// test database, fully migrated and ready for you to query.
func NewDB(t *testing.T) *sql.DB {
  // Call t.Helper() to make sure that any error logs in a failing tests are
  // shown with the appropriate file/line information.
  t.Helper()

  // This configuration will work with the docker-compose.yml file earlier,
  // and assumes that when you run `go test ...` you have a Postgres server
  // running. 
  conf := testdb.Config{
    User:     "postgres",
    Password: "password",
    Host:     "localhost",
    Port:     "5433",
    Options:  "sslmode=disable",
  }

  // You'll need to either implement your own or use one of our adapters for
  // existing migration tools (see the docs below for more information.)
  var m testdb.Migrator = &someImplementation{}

  return testdb.New(t, conf, m)
}
```

Then, in each test, you can just call that helper. You'll either have a valid
`*sql.DB` connection to a test database, or the test will fail and stop
executing.

```go
func TestAQuery(t *testing.T) {
  t.Parallel()
  db := testhelpers.NewDB(t) // this is the helper we just defined above
  ctx := context.Background()

  var result string
  err := db.QueryRowContext(ctx, "SELECT 'hello world'").Scan(&result)
  check.Nil(t, err)
  check.Equal(t, "hello world", result)
}
```

You'll need a Postgres server running or the test will fail with a
complaint that it could not connect to the server to create a new database:

```
--- FAIL: TestAQuery (0.00s)
    /Users/pd/code/example/example_test.go:170: failed to provision test database template: failed to connect to `host=localhost user=postgres database=`: dial error (dial tcp [fe80::1%lo0]:5433: connect: connection refused)
```

## `testdb.New`

This is the only method that `testdb` exposes, and it's the only method you need
to call in your tests. Each time it is called, it:

- Connects to a running Postgres server using the provided config
- Calls `Hash()` on the provided migrator to determine the name of the template
  database.
- If the template database does not exist:
  - Creates a new, empty, database.
  - Gets-or-creates a role `USER=testuser PASSWORD=testpassword` that has full
    ownership and all prvileges on the database, its schemas, tables,
    and sequences.
  - Calls `Prepare()` on the provided migrator to perform any pre-migration
    preparation, like installing extensions or creating roles.
  - Calls `Migrate()` on the provided migrator to actually migrate the database schema.
  - Marks the database as a template
- Creates a new database from the template
- Calls `Verify()` on the provided migrator to confirm that the new test database
  is in the correct state.

It will use both golang-level locks (`sync.Once`) and Postgres-level [advisory locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS) to synchronize, meaning that your migrations will only run one time no matter how many tests, or how many packages, you're testing at the same time.

Once it creates your brand new fresh test database, it will `t.Log()` the connection string so that iff your test fails you can connect to the database and figure out what happened.

## `testdb.Config`

The `Config` struct contains the details needed to connect to a postgres server.
Make sure to connect with a user that has the necessary permissions to create
new databases and roles. Most likely you want to connect as the default
`postgres` user, since you'll be connecting to a dedicated testing-only Postgres
server as described earlier.

```go
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	Options  string
}
```

## `testdb.Migrator`

The `Migrator` interface contains all of the logic needed to prepare a template
database that can be cloned for each of your tests. testdb requires you to
supply a `Migrator` to work. We provide a few for the most popular migration
frameworks:

- [golangmigrator](migrators/golangmigrator/) for [golang-migrate/migrate](https://github.com/golang-migrate/migrate)
- [goosemigrator](migrators/goosemigrator/) for [pressly/goose](https://github.com/pressly/goose)
- [dbmatemigrator](migrators/dbmatemigrator/) for [amacneil/dbmate](https://github.com/amacneil/dbmate)
- [atlasmigrator](migrators/atlasmigrator/) for [ariga/atlas](https://github.com/ariga/atlas)
- [sqlmigrator](#) for [rubenv/sql-migrate](https://github.com/rubenv/sql-migrate)
- 🚧 [pgmigrator](#) for [peterldowns/migrate](https://github.com/peterldowns/migrate)

You can also write your own. The interface is relatively simple, only `Hash()`
and `Migrate()` need to actually do something:

```go
// A Migrator is necessary to provision and verify the database that will be used as as template
// for each test.
type Migrator interface {
	// Hash should return a unique identifier derived from the state of the database
	// after it has been fully migrated. For instance, it may return a hash of all
	// of the migration names and contents.
	//
	// testdb will use the returned Hash to identify a template database. If a
	// Migrator returns a Hash that has already been used to create a template
	// database, it is assumed that the database need not be recreated since it
	// would result in the same schema and data.
	Hash() (string, error)

	// Prepare should perform any plugin or extension installations necessary to
	// make the database ready for the migrations. For instance, you may want to
	// enable certain extensions like `trigram` or `pgcrypto`, or creating or
	// altering certain roles and permissions.
	// Prepare will be given a *sql.DB connected to the template database.
	Prepare(context.Context, *sql.DB, Config) error

	// Migrate is a function that actually performs the schema and data
	// migrations to provision a template database. The connection given to this
	// function is to an entirely new, empty, database. Migrate will be called
	// only once, when the template database is being created.
	Migrate(context.Context, *sql.DB, Config) error

	// Verify is called each time you ask for a new test database instance. It
	// should be cheaper than the call to Migrate(), and should return nil iff
	// the database is in the correct state. An example implementation would be
	// to check that all the migrations have been marked as applied, and
	// otherwise return an error.
	Verify(context.Context, *sql.DB, Config) error
}
```
If you're writing your own `Migrator`, I recommend you use the existing ones
as examples. Most migrators need to do some kind of file/directory hashing in
order to implement `Hash()` &mdash; you may want to use
[the helpers in the `common` subpackage](migrators/common):

- `common.HashFiles(base fs.FS, paths ...string) (string, error)`
- `common.HashDirs(base fs.FS, pattern string, dirs ...string) (string, error)`
- `common.HashDir(pathToDir string) (string, error)`
- `common.HashFile(pathToFile string) (string, error)`
- `common.Execute(ctx context.Context, stdin io.Reader, program string, args ...string) (string, error)`

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

~500ms to prepare the template database one time, then ~10ms to get a new clone
of that template. The time to prepare the template depends on your migration
strategy and your database schema; the time to get a new clone is pretty
constant. It all depends on the speed of you server.

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

## Does this mean I should stop writing unit tests and doing dependency injection?
No! Please keep writing unit tests and doing dependency injection and mocking
and all the other things that make your code well-organized and easily
testable.

This project exists because the database is probably _not_ one of the things
that you want to be mocking in your tests, and most modern applications have a
large amount of logic in Postgres that is hard to mock anyway.

## How does this play out in a real company?
At [Pipe](https://pipe.com), we had thousands of tests using a similar package
to get template-based databases for each test. The whole test suite an in under
a few minutes on reasonably-priced CI machines, and individual packages/tests
ran fast enough on local development machines that developers were happy to add
new database-backed tests without worrying about the cost.

I believe that testdb and a ram-backed Postgres server is fast enough to be
worth it.  If you try it out and don't think so, please open an issue &mdash; I'd be
very interested to see if we can make it work for you, too.

## How can I contribute?

testdb is a standard golang project, you'll need a working golang environment.
If you're of the nix persuasion, this repo comes with a flakes-compatible
development shell that you can enter with `nix develop` (flakes) or `nix-shell`
(standard).

If you use VSCode, the repo comes with suggested extensions and settings.

Testing and linting scripts are defined with Just, see the Justfile to see how
to run those commands manually. There are also Github Actions that will lint and test
your PRs.

Contributions are more than welcome!
