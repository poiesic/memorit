package reembed

import "errors"

var (
	// ErrInvalidMaxAttempts is returned when maxAttempts is <= 0
	ErrInvalidMaxAttempts = errors.New("maxAttempts must be greater than 0")
)
