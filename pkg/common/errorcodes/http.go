package errorcodes

import "errors"

// ToHTTP converts any error into HTTP status and response-safe AppError.
func ToHTTP(err error) (int, AppError) {
	if err == nil {
		return 200, AppError{}
	}

	var appErr AppError
	if errors.As(err, &appErr) {
		if appErr.Status <= 0 {
			fallback := ErrInternal.WithDetails(appErr.Details)
			return fallback.Status, fallback
		}
		return appErr.Status, appErr
	}

	fallback := ErrInternal
	return fallback.Status, fallback
}
