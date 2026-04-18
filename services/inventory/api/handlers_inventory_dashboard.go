package api

import (
	"net/http"

	"github.com/KingWahid/inventory/backend/pkg/common/httpresponse"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
	"github.com/labstack/echo/v4"
)

// GetApiV1InventoryDashboardSummary handles GET /api/v1/inventory/dashboard/summary.
func (h *ServerHandler) GetApiV1InventoryDashboardSummary(c echo.Context) error {
	ctx := c.Request().Context()
	s, err := h.svc.GetDashboardSummary(ctx)
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	data := stub.DashboardSummary{
		TotalProducts:   s.TotalProducts,
		TotalWarehouses: s.TotalWarehouses,
		MovementsToday:  s.MovementsToday,
		LowStockCount:   s.LowStockCount,
	}
	return httpresponse.OK(c, http.StatusOK, data)
}
