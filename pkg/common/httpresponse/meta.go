package httpresponse

import (
	"github.com/labstack/echo/v4"
)

// MetaFromEcho builds meta.request_id from Echo response or request header (same order as errorcodes.WriteHTTPError).
func MetaFromEcho(c echo.Context) map[string]interface{} {
	reqID := c.Response().Header().Get(echo.HeaderXRequestID)
	if reqID == "" {
		reqID = c.Request().Header.Get(echo.HeaderXRequestID)
	}
	if reqID == "" {
		return nil
	}
	return map[string]interface{}{"request_id": reqID}
}
