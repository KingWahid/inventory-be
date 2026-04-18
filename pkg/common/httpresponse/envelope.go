package httpresponse

import (
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/labstack/echo/v4"
)

// APISuccessEnvelope is the §9 success JSON wrapper (success=true).
type APISuccessEnvelope struct {
	Success bool                   `json:"success"`
	Data    any                    `json:"data"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// OK writes a §9 success response with data and optional meta.request_id.
func OK(c echo.Context, status int, data any) error {
	meta := MetaFromEcho(c)
	return c.JSON(status, APISuccessEnvelope{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// OKList writes a §9 success list response: data is the array (or slice), meta includes request_id and pagination.
func OKList(c echo.Context, status int, data any, pg PaginationMeta) error {
	meta := MetaFromEcho(c)
	if meta == nil {
		meta = make(map[string]interface{}, 2)
	}
	meta["pagination"] = pg
	return c.JSON(status, APISuccessEnvelope{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Fail delegates to the shared §9 error envelope (errorcodes.WriteHTTPError).
func Fail(c echo.Context, err error) error {
	return errorcodes.WriteHTTPError(c, err)
}
