package usageitems

import "fmt"

// ErrorKind classifies a usage item error so callers can map it to a status.
type ErrorKind string

const (
	// ErrorInvalid is a rejected field value (bad kind, missing home, ...).
	ErrorInvalid ErrorKind = "invalid"
	// ErrorNotFound is an unknown item id.
	ErrorNotFound ErrorKind = "not_found"
	// ErrorConflict is an id that already exists.
	ErrorConflict ErrorKind = "conflict"
)

// ItemError is a user-facing usage item error.
type ItemError struct {
	Kind    ErrorKind
	Message string
}

func (e *ItemError) Error() string { return e.Message }

func invalidf(format string, args ...any) error {
	return &ItemError{Kind: ErrorInvalid, Message: fmt.Sprintf(format, args...)}
}

func notFoundf(format string, args ...any) error {
	return &ItemError{Kind: ErrorNotFound, Message: fmt.Sprintf(format, args...)}
}

func conflictf(format string, args ...any) error {
	return &ItemError{Kind: ErrorConflict, Message: fmt.Sprintf(format, args...)}
}
