// multierr is a reimplementation of https://github.com/uber-go/multierr/ with a
// much simpler API and zero dependencies. It is designed for internal use only.
package multierr

import (
	"errors"
	"strings"
)

// Join combines multiple errors into a single error that contains all of their
// messages, separated with newlines. Any <nil> errors will be excluded. If all
// the errors are <nil>, a <nil> error is returned.
//
// The resulting error implements Error(), Is(), As(), and Unwrap()
func Join(errs ...error) error {
	var merrs []error
	for _, err := range errs {
		if err == nil {
			continue
		}
		if asMerr, ok := err.(*multierr); ok { //nolint: errorlint // ignore
			merrs = append(merrs, asMerr.Unwrap()...)
		} else {
			merrs = append(merrs, err)
		}
	}
	if len(merrs) == 0 {
		return nil
	}
	return &multierr{errs: merrs}
}

const separator = "\n"

type multierr struct {
	errs []error
}

func (m *multierr) Error() string {
	if m == nil {
		return ""
	}
	var msgs []string
	for _, e := range m.errs {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, separator)
}

func (m *multierr) Is(target error) bool {
	if m == nil {
		return false
	}
	for _, e := range m.errs {
		if errors.Is(e, target) {
			return true
		}
	}
	return false
}

func (m *multierr) As(target any) bool {
	if m == nil {
		return false
	}
	for _, e := range m.errs {
		if errors.As(e, target) {
			return true
		}
	}
	return false
}

func (m *multierr) Unwrap() []error {
	if m == nil {
		return nil
	}
	return m.errs
}
