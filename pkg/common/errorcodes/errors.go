// Package errorcodes defines domain-facing HTTP errors and the §9 JSON error envelope.
//
// Decision (ADR docs/adr/001-errors.md): extend AppError with chaining instead of a separate
// CustomError type — one classification path through ToHTTP and WriteHTTPError.
package errorcodes

import "errors"

// AppError is the domain-level error contract returned from services and mapped to HTTP JSON.
type AppError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	MessageID string         `json:"message_id,omitempty"` // stable id for future i18n / clients
	Status    int            `json:"-"`
	Details   map[string]any `json:"details,omitempty"`
}

func (e AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

// Is implements error matching so errors.Is works with WithDetails/WithMessageID clones sharing the same Code.
func (e AppError) Is(target error) bool {
	if target == nil {
		return false
	}
	var t AppError
	if !errors.As(target, &t) {
		return false
	}
	return e.Code != "" && e.Code == t.Code
}

// New constructs an AppError (machine code, safe message, HTTP status).
func New(code, message string, status int) AppError {
	return AppError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

func (e AppError) WithDetails(details map[string]any) AppError {
	next := e
	if len(details) == 0 {
		next.Details = nil
		return next
	}

	next.Details = make(map[string]any, len(details))
	for k, v := range details {
		next.Details[k] = v
	}
	return next
}

// WithMessageID sets a stable message key for translations (optional).
func (e AppError) WithMessageID(id string) AppError {
	e.MessageID = id
	return e
}

// WithStatus replaces HTTP status (use when cloning a template error with a different code).
func (e AppError) WithStatus(status int) AppError {
	e.Status = status
	return e
}

// WithCode replaces the machine-readable code (keeps message and status unless you chain further).
func (e AppError) WithCode(code string) AppError {
	e.Code = code
	return e
}

// Problem starts a fluent chain from a human-readable message (defaults to INTERNAL_ERROR / 500).
func Problem(msg string) AppError {
	return AppError{
		Code:    CodeInternalError,
		Message: msg,
		Status:  500,
	}
}
