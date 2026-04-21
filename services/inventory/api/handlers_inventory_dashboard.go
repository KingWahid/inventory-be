package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/KingWahid/inventory/backend/pkg/common/httpresponse"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
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

// GetApiV1InventoryDashboardMovementsChart handles GET /api/v1/inventory/dashboard/movements/chart.
func (h *ServerHandler) GetApiV1InventoryDashboardMovementsChart(c echo.Context, params stub.GetApiV1InventoryDashboardMovementsChartParams) error {
	ctx := c.Request().Context()
	period := ""
	if params.Period != nil {
		period = string(*params.Period)
	}
	out, err := h.svc.GetDashboardMovementsChart(ctx, period)
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	points := make([]stub.DashboardMovementChartPoint, 0, len(out.Points))
	for _, p := range out.Points {
		t, err := time.Parse(time.DateOnly, p.BucketStart)
		if err != nil {
			return httpresponse.Fail(c, errorcodes.ErrInternal)
		}
		points = append(points, stub.DashboardMovementChartPoint{
			BucketStart:   openapi_types.Date{Time: t.UTC()},
			MovementCount: p.MovementCount,
		})
	}
	data := stub.DashboardMovementChart{
		Period: stub.DashboardMovementChartPeriod(out.Period),
		Points: points,
	}
	return httpresponse.OK(c, http.StatusOK, data)
}

// GetDashboardStorageUtilization handles GET /api/v1/inventory/dashboard/storage-utilization.
func (h *ServerHandler) GetDashboardStorageUtilization(c echo.Context) error {
	ctx := c.Request().Context()
	limit := 3
	if raw := c.QueryParam("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return httpresponse.Fail(c, errorcodes.ErrValidationError.WithDetails(map[string]any{
				"message": "limit must be an integer",
			}))
		}
		limit = n
	}
	rows, err := h.svc.GetDashboardStorageUtilization(ctx, limit)
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	return httpresponse.OK(c, http.StatusOK, rows)
}
