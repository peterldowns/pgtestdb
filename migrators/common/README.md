# common

common contains helpers for use in different implementations of the `Migrator` interface.

## Hashing

```go
// HashFiles will return as unique hash based on the contents of the specified
// files. If `base` is nil, it will read the paths from the real file system.
//
// Examples:
//
//	HashFiles(nil, "0001_initial.sql")
//	HashFiles(nil, "0001_initial.sql", "0002_users.up.sql")
//	HashDirs(embeddedFS, "migrations/0001_initial.sql")
func HashFiles(base fs.FS, paths ...string) (string, error)
```

```go
// HashDirs will return as unique hash based on the contents of all files that
// match a given pattern, from any of the given directories. If `base` is nil,
// it will read the directories and files from the real file system.
//
// Examples:
//
//	HashDirs(nil, "*.sql", "migrations")
//	HashDirs(nil, "*.sql", "migrations/old", "migrations/current")
//	HashDirs(embeddedFS, "*.sql", ".")
func HashDirs(base fs.FS, pattern string, dirs ...string) (string, error)
```

```go
// Returns a unique hash based on the contents of any "*.sql" files found in the
// specified directory.
func HashDir(pathToDir string) (string, error)
```

```go
// Returns a unique hash based on the contents of the specified file.
func HashFile(pathToFile string) (string, error)
```

```go
// NewRecursiveHash creates a new [RecursiveHash], and adds any of the given fields
// to it.
//
// Examples:
//
//  hash, _ := NewRecursiveHash()
//  _ = hash.AddField(Field("CreateMigrationsTable", settings.CreateMigrationsTable))
//  _ = hash.AddField(Field("MigrationsTableName", settings.MigrationsTableName))
//  _ = hash.Add([]byte("hello"))
//  _ = hash.Add([]byte("world"))
//  out, _ := hash.String()
func NewRecursiveHash(fields ...HashField) (RecursiveHash, error)
```

## Shelling out
```go
// Execute shells out to a `program`, passing it STDIN (if given) and any specified arguments.
//
// Examples:
//
//	Execute(ctx, nil, "echo", "hello", "world"
//	Execute(ctx, nil, "bash", "-c", "echo 'hello world'"
func Execute(ctx context.Context, stdin io.Reader, program string, args ...string) (string, error)
```