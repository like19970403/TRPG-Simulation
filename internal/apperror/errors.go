package apperror

import "errors"

// Sentinel errors returned by repository layers.
// Handlers use errors.Is() to map these to HTTP status codes.
var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate")
)
