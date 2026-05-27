package errorcodes

import (
	"go.uber.org/zap"

	"github.com/labstack/echo/v4"
)

// ErrorPayload is the §9 "error" object (ARCHITECTURE standard response format).
type ErrorPayload struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	MessageID string         `json:"message_id,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

// APIErrorEnvelope is the §9 error JSON wrapper (success=false).
type APIErrorEnvelope struct {
	Success bool                   `json:"success"`
	Err     *ErrorPayload          `json:"error"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// WriteHTTPError maps err through ToHTTP and writes the §9 envelope with optional request_id in meta.
func WriteHTTPError(c echo.Context, err error) error {
	if err == nil {
		return nil
	}
	status, ae := ToHTTP(err)

	// Log all 5xx errors with the original error for debugging.
	if status >= 500 {
		reqID := c.Response().Header().Get(echo.HeaderXRequestID)
		zap.L().Error("internal server error",
			zap.Error(err),
			zap.String("request_id", reqID),
			zap.String("method", c.Request().Method),
			zap.String("uri", c.Request().RequestURI),
		)
	}

	reqID := c.Response().Header().Get(echo.HeaderXRequestID)
	if reqID == "" {
		reqID = c.Request().Header.Get(echo.HeaderXRequestID)
	}

	env := APIErrorEnvelope{
		Success: false,
		Err: &ErrorPayload{
			Code:      ae.Code,
			Message:   ae.Message,
			MessageID: ae.MessageID,
			Details:   ae.Details,
		},
	}
	if reqID != "" {
		env.Meta = map[string]interface{}{"request_id": reqID}
	}
	return c.JSON(status, env)
}
