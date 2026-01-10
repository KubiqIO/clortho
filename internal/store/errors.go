package store

import "errors"

// Common store errors for use with errors.Is()
var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate entry")
)
