package api

import (
	"bytes"
	"io"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/KingWahid/inventory/backend/pkg/common/httpresponse"
	"github.com/KingWahid/inventory/backend/pkg/idempotency"
	movementuc "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/usecase"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
	"github.com/labstack/echo/v4"
)

func ptrStringFromEnumType(p *stub.GetApiV1InventoryMovementsParamsType) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

func ptrStringFromEnumStatus(p *stub.GetApiV1InventoryMovementsParamsStatus) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

func ptrStringFromSort(p *stub.Sort) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

func ptrStringFromOrderMovement(p *stub.GetApiV1InventoryMovementsParamsOrder) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

func uuidPtrToStringPtr(u *openapi_types.UUID) *string {
	if u == nil {
		return nil
	}
	s := uuid.UUID(*u).String()
	return &s
}

func linesFromStub(lines []stub.MovementLineCreate) []movementuc.LineInput {
	out := make([]movementuc.LineInput, 0, len(lines))
	for i := range lines {
		out = append(out, movementuc.LineInput{
			ProductID: uuid.UUID(lines[i].ProductId).String(),
			Quantity:  lines[i].Quantity,
			Notes:     lines[i].Notes,
		})
	}
	return out
}

func readBindRawBodyMovementHash(c echo.Context, dst any) (string, error) {
	raw, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return "", err
	}
	c.Request().Body = io.NopCloser(bytes.NewReader(raw))
	if err := c.Bind(dst); err != nil {
		return "", err
	}
	return idempotency.SHA256Hex(raw), nil
}

// GetApiV1InventoryMovements handles GET /api/v1/inventory/movements.
func (h *ServerHandler) GetApiV1InventoryMovements(c echo.Context, params stub.GetApiV1InventoryMovementsParams) error {
	ctx := c.Request().Context()
	page, perPage := resolvePagePerPage(c, params.Page, params.PerPage)
	search := ""
	if params.Search != nil {
		search = string(*params.Search)
	}

	out, err := h.svc.ListMovements(ctx, movementuc.ListMovementsInput{
		Page:    &page,
		PerPage: &perPage,
		Type:    ptrStringFromEnumType(params.Type),
		Status:  ptrStringFromEnumStatus(params.Status),
		Search:  &search,
		Sort:    ptrStringFromSort(params.Sort),
		Order:   ptrStringFromOrderMovement(params.Order),
	})
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	data := make([]stub.Movement, 0, len(out.Items))
	for i := range out.Items {
		row, mErr := movementRepoToStub(out.Items[i])
		if mErr != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, mErr.Error())
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

// GetApiV1InventoryMovementsMovementId handles GET /api/v1/inventory/movements/{movementId}.
func (h *ServerHandler) GetApiV1InventoryMovementsMovementId(c echo.Context, movementId stub.MovementId) error {
	ctx := c.Request().Context()
	m, err := h.svc.GetMovement(ctx, uuid.UUID(movementId).String())
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := movementRepoToStub(m)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return httpresponse.OK(c, http.StatusOK, row)
}

// PostApiV1InventoryMovementsInbound handles POST /api/v1/inventory/movements/inbound.
func (h *ServerHandler) PostApiV1InventoryMovementsInbound(c echo.Context, params stub.PostApiV1InventoryMovementsInboundParams) error {
	ctx := c.Request().Context()
	var body stub.InboundMovementCreateRequest
	bodyHash, err := readBindRawBodyMovementHash(c, &body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	dest := uuid.UUID(body.DestinationWarehouseId).String()
	m, err := h.svc.CreateInbound(ctx, dest, movementuc.CreateMovementBase{
		ReferenceNumber:      body.ReferenceNumber,
		Notes:                body.Notes,
		IdempotencyKey:       string(params.IdempotencyKey),
		RequestHashSHA256Hex: bodyHash,
		Lines:                linesFromStub(body.Lines),
	})
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := movementRepoToStub(m)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return httpresponse.OK(c, http.StatusCreated, row)
}

// PostApiV1InventoryMovementsOutbound handles POST /api/v1/inventory/movements/outbound.
func (h *ServerHandler) PostApiV1InventoryMovementsOutbound(c echo.Context, params stub.PostApiV1InventoryMovementsOutboundParams) error {
	ctx := c.Request().Context()
	var body stub.OutboundMovementCreateRequest
	bodyHash, err := readBindRawBodyMovementHash(c, &body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	src := uuid.UUID(body.SourceWarehouseId).String()
	m, err := h.svc.CreateOutbound(ctx, src, movementuc.CreateMovementBase{
		ReferenceNumber:      body.ReferenceNumber,
		Notes:                body.Notes,
		IdempotencyKey:       string(params.IdempotencyKey),
		RequestHashSHA256Hex: bodyHash,
		Lines:                linesFromStub(body.Lines),
	})
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := movementRepoToStub(m)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return httpresponse.OK(c, http.StatusCreated, row)
}

// PostApiV1InventoryMovementsTransfer handles POST /api/v1/inventory/movements/transfer.
func (h *ServerHandler) PostApiV1InventoryMovementsTransfer(c echo.Context, params stub.PostApiV1InventoryMovementsTransferParams) error {
	ctx := c.Request().Context()
	var body stub.TransferMovementCreateRequest
	bodyHash, err := readBindRawBodyMovementHash(c, &body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	src := uuid.UUID(body.SourceWarehouseId).String()
	dst := uuid.UUID(body.DestinationWarehouseId).String()
	m, err := h.svc.CreateTransfer(ctx, src, dst, movementuc.CreateMovementBase{
		ReferenceNumber:      body.ReferenceNumber,
		Notes:                body.Notes,
		IdempotencyKey:       string(params.IdempotencyKey),
		RequestHashSHA256Hex: bodyHash,
		Lines:                linesFromStub(body.Lines),
	})
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := movementRepoToStub(m)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return httpresponse.OK(c, http.StatusCreated, row)
}

// PostApiV1InventoryMovementsAdjustment handles POST /api/v1/inventory/movements/adjustment.
func (h *ServerHandler) PostApiV1InventoryMovementsAdjustment(c echo.Context, params stub.PostApiV1InventoryMovementsAdjustmentParams) error {
	ctx := c.Request().Context()
	var body stub.AdjustmentMovementCreateRequest
	bodyHash, err := readBindRawBodyMovementHash(c, &body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	m, err := h.svc.CreateAdjustment(ctx, uuidPtrToStringPtr(body.SourceWarehouseId), uuidPtrToStringPtr(body.DestinationWarehouseId), movementuc.CreateMovementBase{
		ReferenceNumber:      body.ReferenceNumber,
		Notes:                body.Notes,
		IdempotencyKey:       string(params.IdempotencyKey),
		RequestHashSHA256Hex: bodyHash,
		Lines:                linesFromStub(body.Lines),
	})
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := movementRepoToStub(m)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return httpresponse.OK(c, http.StatusCreated, row)
}

// PostApiV1InventoryMovementsMovementIdConfirm handles POST .../confirm.
func (h *ServerHandler) PostApiV1InventoryMovementsMovementIdConfirm(c echo.Context, movementId stub.MovementId) error {
	ctx := c.Request().Context()
	m, err := h.svc.ConfirmMovement(ctx, uuid.UUID(movementId).String())
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := movementRepoToStub(m)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return httpresponse.OK(c, http.StatusOK, row)
}

// PostApiV1InventoryMovementsMovementIdCancel handles POST .../cancel.
func (h *ServerHandler) PostApiV1InventoryMovementsMovementIdCancel(c echo.Context, movementId stub.MovementId) error {
	ctx := c.Request().Context()
	m, err := h.svc.CancelMovement(ctx, uuid.UUID(movementId).String())
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := movementRepoToStub(m)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return httpresponse.OK(c, http.StatusOK, row)
}
