package errorcodes

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// ToHTTP converts any error into HTTP status and a response-safe AppError (flat; use WriteHTTPError for §9 envelope).
func ToHTTP(err error) (int, AppError) {
	if err == nil {
		return 200, AppError{}
	}

	var he *echo.HTTPError
	if errors.As(err, &he) {
		return fromEchoHTTPError(he)
	}

	var ae AppError
	if errors.As(err, &ae) {
		if ae.Status <= 0 {
			fallback := ErrInternal.WithDetails(ae.Details)
			if ae.MessageID != "" {
				fallback = fallback.WithMessageID(ae.MessageID)
			}
			return fallback.Status, fallback
		}
		return ae.Status, ae
	}

	if app, ok := mapKnownSentinels(err); ok {
		return app.Status, app
	}

	return ErrInternal.Status, ErrInternal
}

func fromEchoHTTPError(he *echo.HTTPError) (int, AppError) {
	code := he.Code
	if code == 0 {
		code = http.StatusInternalServerError
	}
	msg := echoMessage(he)

	switch code {
	case http.StatusBadRequest:
		return code, ErrValidationError.WithDetails(map[string]any{"message": msg})
	case http.StatusUnauthorized:
		return code, ErrUnauthorized.WithDetails(map[string]any{"message": msg})
	case http.StatusForbidden:
		return code, ErrForbidden.WithDetails(map[string]any{"message": msg})
	case http.StatusNotFound:
		return code, ErrNotFound.WithDetails(map[string]any{"message": msg})
	case http.StatusConflict:
		return code, ErrConflict.WithDetails(map[string]any{"message": msg})
	case http.StatusNotImplemented:
		return code, ErrNotImplemented.WithDetails(map[string]any{"message": msg})
	default:
		if code >= 400 && code < 500 {
			return code, New(CodeValidationError, msg, code).WithDetails(map[string]any{"message": msg})
		}
		if code >= 500 {
			return code, ErrInternal
		}
		return http.StatusInternalServerError, ErrInternal
	}
}

func echoMessage(he *echo.HTTPError) string {
	switch m := he.Message.(type) {
	case string:
		if m != "" {
			return m
		}
	case error:
		return m.Error()
	case fmt.Stringer:
		return m.String()
	}
	return http.StatusText(he.Code)
}

func mapKnownSentinels(err error) (AppError, bool) {
	switch {
	case errors.Is(err, ErrJWTParseToken),
		errors.Is(err, ErrJWTInvalidClaims),
		errors.Is(err, ErrJWTInvalidSigning),
		errors.Is(err, ErrJWTInvalidTokenType):
		return ErrUnauthorized, true
	case errors.Is(err, ErrJWTInvalidSubject),
		errors.Is(err, ErrJWTInvalidTenantID):
		return ErrValidationError, true
	case errors.Is(err, ErrTenantContextMissing):
		return ErrUnauthorized, true
	case errors.Is(err, ErrJWTInvalidSecret),
		errors.Is(err, ErrJWTInvalidTTL):
		return ErrInternal, true
	default:
		return AppError{}, false
	}
}
