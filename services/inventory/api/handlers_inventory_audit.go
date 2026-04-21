package api

import (
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/KingWahid/inventory/backend/pkg/common/httpresponse"
	audituc "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/usecase"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
	"github.com/labstack/echo/v4"
)

func (h *ServerHandler) GetApiV1InventoryAuditLogs(c echo.Context, params stub.GetApiV1InventoryAuditLogsParams) error {
	ctx := c.Request().Context()
	page, perPage := resolvePagePerPage(c, params.Page, params.PerPage)

	in := audituc.ListAuditLogsInput{
		Page:        &page,
		PerPage:     &perPage,
		Entity:      params.Entity,
		Action:      params.Action,
		CreatedFrom: params.CreatedFrom,
		CreatedTo:   params.CreatedTo,
	}
	if params.EntityId != nil {
		s := uuid.UUID(*params.EntityId).String()
		in.EntityID = &s
	}
	if params.UserId != nil {
		s := uuid.UUID(*params.UserId).String()
		in.UserID = &s
	}

	out, err := h.svc.ListAuditLogs(ctx, in)
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	data := make([]stub.AuditLog, 0, len(out.Items))
	for i := range out.Items {
		row, convErr := auditEntryToStub(out.Items[i])
		if convErr != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, convErr.Error())
		}
		data = append(data, row)
	}
	pg := httpresponse.PaginationMeta{
		Page:       out.Page,
		PerPage:    out.PerPage,
		Total:      out.Total,
		TotalPages: httpresponse.ComputeTotalPages(out.Total, int64(out.PerPage)),
	}
	return httpresponse.OKList(c, http.StatusOK, data, pg)
}

func (h *ServerHandler) GetApiV1InventoryAuditLogsAuditEntityAuditEntityId(c echo.Context, auditEntity string, auditEntityId openapi_types.UUID, params stub.GetApiV1InventoryAuditLogsAuditEntityAuditEntityIdParams) error {
	ctx := c.Request().Context()
	page, perPage := resolvePagePerPage(c, params.Page, params.PerPage)
	entity := auditEntity
	eid := uuid.UUID(auditEntityId).String()

	in := audituc.ListAuditLogsInput{
		Page:     &page,
		PerPage:  &perPage,
		Entity:   &entity,
		EntityID: &eid,
	}
	out, err := h.svc.ListAuditLogs(ctx, in)
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	data := make([]stub.AuditLog, 0, len(out.Items))
	for i := range out.Items {
		row, convErr := auditEntryToStub(out.Items[i])
		if convErr != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, convErr.Error())
		}
		data = append(data, row)
	}
	pg := httpresponse.PaginationMeta{
		Page:       out.Page,
		PerPage:    out.PerPage,
		Total:      out.Total,
		TotalPages: httpresponse.ComputeTotalPages(out.Total, int64(out.PerPage)),
	}
	return httpresponse.OKList(c, http.StatusOK, data, pg)
}
