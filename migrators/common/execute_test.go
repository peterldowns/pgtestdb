package common_test

import (
	"context"
	"strings"
	"testing"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgtestdb/migrators/common"
)

func TestExecute(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	resp, err := common.Execute(ctx, nil, "echo", "hello world")
	assert.Nil(t, err)
	check.Equal(t, "hello world", resp)
}

func TestExecuteQuoting(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	resp, err := common.Execute(ctx, nil, "bash", "-c", "echo 'hello world'")
	assert.Nil(t, err)
	check.Equal(t, "hello world", resp)
}

func TestExecuteStdin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	resp, err := common.Execute(ctx, strings.NewReader("hello world"), "cat")
	assert.Nil(t, err)
	check.Equal(t, "hello world", resp)
}

func TestExecuteStripsOneTrailingNewline(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	resp, err := common.Execute(ctx, strings.NewReader("hello world\t\n\n"), "cat")
	assert.Nil(t, err)
	check.Equal(t, string("hello world\t\n"), resp)
}
