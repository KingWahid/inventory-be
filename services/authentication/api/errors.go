package api

import (
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func httpErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	if err == nil {
		return
	}
	status, _ := errorcodes.ToHTTP(err)
	if status >= 500 && !errorcodes.IsClassified(err) {
		zap.L().Warn("unclassified server error", zap.Error(err))
	}
	_ = errorcodes.WriteHTTPError(c, err)
}
