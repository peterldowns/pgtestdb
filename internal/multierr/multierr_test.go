package multierr_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgtestdb/internal/multierr"
)

func TestJoinNils(t *testing.T) {
	t.Parallel()
	t.Run("empty join returns nil error", func(t *testing.T) {
		t.Parallel()
		check.Nil(t, multierr.Join())
	})
	t.Run("join with one nil returns nil error", func(t *testing.T) {
		t.Parallel()
		check.Nil(t, multierr.Join(nil))
	})
	t.Run("join with multiple nils returns nil error", func(t *testing.T) {
		t.Parallel()
		check.Nil(t, multierr.Join(nil, nil, nil, nil))
	})
}

func TestJoinErrorsWithNils(t *testing.T) {
	t.Parallel()
	t.Run("error with nil returns error", func(t *testing.T) {
		t.Parallel()
		example := fmt.Errorf("example error")
		res := multierr.Join(example, nil, nil)
		check.Equal(t, example, res, cmpopts.EquateErrors())
	})
	t.Run("nil with error returns error", func(t *testing.T) {
		t.Parallel()
		example := fmt.Errorf("example error")
		res := multierr.Join(nil, nil, example)
		check.Equal(t, example, res, cmpopts.EquateErrors())
	})
}

func TestJoinErrors(t *testing.T) {
	t.Parallel()
	t.Run("merge two errors returns a multierr", func(t *testing.T) {
		t.Parallel()
		a := fmt.Errorf("error a")
		b := fmt.Errorf("error b")
		res := multierr.Join(nil, a, nil, b, nil)
		check.Equal(t, "error a\nerror b", res.Error())
	})
	t.Run("merge an error with a multierr returns a multierr", func(t *testing.T) {
		t.Parallel()
		me := multierr.Join(
			fmt.Errorf("error a"),
			fmt.Errorf("error b"),
		)
		c := fmt.Errorf("error c")
		res := multierr.Join(nil, me, nil, c, nil)
		check.Equal(t, "error a\nerror b\nerror c", res.Error())
	})
	t.Run("merge a multierr with a multierr returns a multierr", func(t *testing.T) {
		t.Parallel()
		me := multierr.Join(
			fmt.Errorf("error a"),
			fmt.Errorf("error b"),
		)
		mf := multierr.Join(
			fmt.Errorf("error c"),
			fmt.Errorf("error d"),
		)
		res := multierr.Join(nil, me, nil, mf, nil)
		check.Equal(t, "error a\nerror b\nerror c\nerror d", res.Error())
	})
}
