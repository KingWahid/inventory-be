package errorcodes

import (
	"errors"

	"github.com/labstack/echo/v4"
)

// IsClassified reports whether ToHTTP will treat err as domain-classified (not fall through to INTERNAL only).
func IsClassified(err error) bool {
	if err == nil {
		return true
	}
	var ae AppError
	if errors.As(err, &ae) {
		return true
	}
	var he *echo.HTTPError
	if errors.As(err, &he) {
		return true
	}
	_, ok := mapKnownSentinels(err)
	return ok
}
