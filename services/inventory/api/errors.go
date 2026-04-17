package api

import (
	"github.com/labstack/echo/v4"

	"github.com/your-org/inventory/backend/pkg/common/errorcodes"
)

func httpErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	code, body := errorcodes.ToHTTP(err)
	if err == nil {
		return
	}
	_ = c.JSON(code, body)
}
