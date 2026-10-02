// Package apperr defines CLI errors that carry an explicit process exit code.
package apperr

import (
	"errors"
	"fmt"
)

// Process exit codes used by the CLI.
const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

// Error is an error with an associated process exit code.
type Error struct {
	Message string
	Code    int
	Err     error
}

func (e *Error) Error() string {
	switch {
	case e.Message == "" && e.Err != nil:
		return e.Err.Error()
	case e.Err != nil:
		return e.Message + ": " + e.Err.Error()
	default:
		return e.Message
	}
}

// Unwrap exposes the wrapped error to errors.Is/errors.As.
func (e *Error) Unwrap() error { return e.Err }

// New returns an operational error (exit code 1).
func New(format string, args ...any) *Error {
	return &Error{Message: fmt.Sprintf(format, args...), Code: ExitError}
}

// Usage returns a usage error (exit code 2).
func Usage(format string, args ...any) *Error {
	return &Error{Message: fmt.Sprintf(format, args...), Code: ExitUsage}
}

// Wrap annotates err as an operational error.
func Wrap(err error, format string, args ...any) *Error {
	return &Error{Message: fmt.Sprintf(format, args...), Err: err, Code: ExitError}
}

// WrapUsage annotates err as a usage error.
func WrapUsage(err error, format string, args ...any) *Error {
	return &Error{Message: fmt.Sprintf(format, args...), Err: err, Code: ExitUsage}
}

// ExitCode reports the process exit code for err.
func ExitCode(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ExitError
}
