package errorcodes

// AppError is a domain-level error contract used across services.
type AppError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Status  int            `json:"-"`
	Details map[string]any `json:"details,omitempty"`
}

func (e AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

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
