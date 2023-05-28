package common

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strings"
)

// HashFiles will return as unique hash based on the contents of the specified
// files. If `base` is nil, it will read the paths from the real file system.
//
// Examples:
//
//	HashFiles(nil, "0001_initial.sql")
//	HashFiles(nil, "0001_initial.sql", "0002_users.up.sql")
//	HashDirs(embeddedFS, "migrations/0001_initial.sql")
func HashFiles(base fs.FS, paths ...string) (string, error) {
	var contents []byte
	var err error
	hash := md5.New()
	for _, path := range paths {
		if base == nil {
			contents, err = os.ReadFile(path)
		} else {
			contents, err = fs.ReadFile(base, path)
		}
		if err != nil {
			return "", err
		}
		fmt.Fprintf(hash, "%x=%x\n", hash.Sum(nil), md5.Sum(contents))
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
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
	hash := md5.New()
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
			fmt.Fprintf(hash, "%x=%x\n", hash.Sum(nil), md5.Sum(contents))
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
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

// Execute shells out to a `program`, passing it STDIN (if given) and any specified arguments.
//
// Examples:
//
//	Execute(ctx, nil, "echo", "hello", "world"
//	Execute(ctx, nil, "bash", "-c", "echo 'hello world'"
func Execute(ctx context.Context, stdin io.Reader, program string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Stdin = stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		if errMsg := stderr.String(); errMsg != "" {
			return "", fmt.Errorf("program %s failed with error(%w): %s", program, err, errMsg)
		}
		return "", fmt.Errorf("program %s failed with error(%w)", program, err)
	}
	return strings.TrimSuffix(stdout.String(), "\n"), nil
}

// TODO: docs
func NewRecursiveHash(fields ...HashField) (RecursiveHash, error) {
	hash := RecursiveHash{md5.New()}
	err := hash.AddFields(fields...)
	return hash, err
}

type RecursiveHash struct {
	hash.Hash
}

func (h RecursiveHash) Add(bytes []byte) error {
	_, err := fmt.Fprintf(h, "%x=%x\n", h.Sum(nil), md5.Sum(bytes))
	return err
}

func (h RecursiveHash) AddField(key string, value any) error {
	return h.Add([]byte(fmt.Sprintf("%s=%v", key, value)))
}

func (h RecursiveHash) AddFields(fields ...HashField) error {
	for _, field := range fields {
		if err := h.AddField(field.Key, field.Value); err != nil {
			return err
		}
	}
	return nil
}

func (h RecursiveHash) String() string {
	return hex.EncodeToString(h.Sum(nil))
}

func Field(key string, value any) HashField {
	return HashField{Key: key, Value: value}
}

type HashField struct {
	Key   string
	Value any
}
