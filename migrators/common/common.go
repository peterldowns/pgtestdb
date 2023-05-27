package common

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strings"
)

// TODO: FS-based helpers for other adapters in the future
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	contents, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	x := md5.Sum(contents)
	return hex.EncodeToString(x[:]), nil
}

// Hashes the directory by rolling up the contents of all the .sql files.
// The filenames are not included in the hash because they do not affect the
// final schema of the database.
func HashDir(path string) (string, error) {
	h := md5.New()
	dirfs := os.DirFS(path)
	entries, err := fs.Glob(dirfs, "*.sql")
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		contents, err := fs.ReadFile(dirfs, entry)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%x=%x\n", h.Sum(nil), md5.Sum(contents))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

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
