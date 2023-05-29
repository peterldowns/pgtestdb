package common

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

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
