package errdefs

import (
	"errors"
	"fmt"
)

var (
	//общие ошибки
	ErrNotFound        = errors.New("not found")
	ErrDB              = errors.New("DB error")
	ErrInvalidInput    = errors.New("invalid input")
	ErrConflict        = errors.New("conflict")
	NotEnoughTokens    = errors.New("not enough free tokens")
	TokensLeCap        = errors.New("Count tokens nore then capacity")
	ErrMigrationFailed = errors.New("Migration failed")
	ErrNoBackends	   = errors.New("Not free backend")
	ErrRateLimitExceeded = errors.New("ErrRateLimitExceeded")
)

// fmt.Errorf с %w
func Wrap(err error, context string) error {
	return fmt.Errorf("%s: %w", context, err)
}

// С форматрованием
func Wrapf(err error, format string, args ...interface{}) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target interface{}) bool {
	return errors.As(err, target)
}
