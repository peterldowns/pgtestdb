package common

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	hashlib "hash"
	"io/fs"
	"os"
)

// NewRecursiveHash creates a new [RecursiveHash], and adds any of the given fields
// to it.
//
// Example:
//
//	hash, _ := NewRecursiveHash()
//	_ = hash.AddField(Field("CreateMigrationsTable", settings.CreateMigrationsTable))
//	_ = hash.AddField(Field("MigrationsTableName", settings.MigrationsTableName))
//	_ = hash.Add([]byte("hello"))
//	_ = hash.Add([]byte("world"))
//	out, _ := hash.String()
func NewRecursiveHash(fields ...HashField) RecursiveHash {
	hash := RecursiveHash{md5.New()}
	hash.AddFields(fields...)
	return hash
}

// RecursiveHash is a small wrapper around an md5 hash. Each time more data is
// added to the hash, it will update itself to include the hash of all previous
// contents. This is good for hashing multiple migration files. The interface is slightly
// easier to use than constructing an md5 hash on your own.
type RecursiveHash struct {
	hashlib.Hash
}

// Add updates the hash with the hash of new content.
func (h RecursiveHash) Add(bytes []byte) {
	if _, err := fmt.Fprintf(h, "%x=%x\n", h.Sum(nil), md5.Sum(bytes)); err != nil {
		panic(err)
	}
}

// AddField updates the hash with the hash of a new field.
func (h RecursiveHash) AddField(key string, value any) {
	h.Add([]byte(fmt.Sprintf("%s=%v", key, value)))
}

// AddFields updates the hash with the hash of multiple fields at once.
func (h RecursiveHash) AddFields(fields ...HashField) {
	for _, field := range fields {
		h.AddField(field.Key, field.Value)
	}
}

func (h RecursiveHash) AddFiles(base fs.FS, paths ...string) error {
	filesHash, err := HashFiles(base, paths...)
	if err != nil {
		return err
	}
	h.Add([]byte(filesHash))
	return nil
}

func (h RecursiveHash) AddDirs(base fs.FS, pattern string, dirs ...string) error {
	dirsHash, err := HashDirs(base, pattern, dirs...)
	if err != nil {
		return err
	}
	h.Add([]byte(dirsHash))
	return nil
}

func (h RecursiveHash) String() string {
	return hex.EncodeToString(h.Sum(nil))
}

// Field is a helper for incorporating certain settings/config values into a
// Hash() function result. You should hash.Add(Field(key, val)) any
// configuration settings that affect the final schema of a database, so that if
// those settings change the database template gets recreated.
func Field(key string, value any) HashField {
	return HashField{Key: key, Value: value}
}

// HashField is a convenience type, you should create them with [Field].
type HashField struct {
	Key   string
	Value any
}

// HashFiles will return as unique hash based on the contents of the specified
// files. If `base` is nil, it will read the paths from the real file system.
//
// Examples:
//
//	HashFiles(nil, "0001_initial.sql")
//	HashFiles(nil, "0001_initial.sql", "0002_users.up.sql")
//	HashDirs(embeddedFS, "migrations/0001_initial.sql")
func HashFiles(base fs.FS, paths ...string) (string, error) {
	var err error
	var contents []byte
	hash := NewRecursiveHash()
	for _, path := range paths {
		if base == nil {
			contents, err = os.ReadFile(path)
		} else {
			contents, err = fs.ReadFile(base, path)
		}
		if err != nil {
			return "", err
		}
		hash.Add(contents)
	}
	return hash.String(), nil
}

// HashDirs will return as unique hash based on the contents of all files that
// match a given pattern, from any of the given directories. If `base` is nil,
// it will read the directories and files from the real file system.
//
// Examples:
//
//	HashDirs(nil, "*.sql", "migrations")
//	HashDirs(nil, "*.sql", "migrations/old", "migrations/current")
//	HashDirs(embeddedFS, "*.sql", ".")
func HashDirs(base fs.FS, pattern string, dirs ...string) (string, error) {
	var dir fs.FS
	hash := NewRecursiveHash()
	for _, path := range dirs {
		if base == nil {
			dir = os.DirFS(path)
		} else {
			var err error
			dir, err = fs.Sub(base, path)
			if err != nil {
				return "", err
			}
		}
		entries, err := fs.Glob(dir, pattern)
		if err != nil {
			return "", err
		}
		for _, entry := range entries {
			contents, err := fs.ReadFile(dir, entry)
			if err != nil {
				return "", err
			}
			hash.Add(contents)
		}
	}
	return hash.String(), nil
}

// Returns a unique hash based on the contents of any "*.sql" files found in the
// specified directory.
func HashDir(pathToDir string) (string, error) {
	return HashDirs(nil, "*.sql", pathToDir)
}

// Returns a unique hash based on the contents of the specified file.
func HashFile(pathToFile string) (string, error) {
	return HashFiles(nil, pathToFile)
}
