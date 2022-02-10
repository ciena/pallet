package podpredicates

import (
	"errors"
)

var (
	// ErrNoPredicate is returned when a predicate is not implemented.
	ErrNoPredicate = errors.New("no predicate found")

	// ErrAlreadyExists is returned when an entry already exists in registry.
	ErrAlreadyExists = errors.New("already exists")
)
