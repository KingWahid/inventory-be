package usecase

import (
	"context"
	"strings"

	cachepkg "github.com/KingWahid/inventory/backend/pkg/cache"
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/pkg/common/pagination"
	"github.com/KingWahid/inventory/backend/pkg/database/transaction"
	flowaudit "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/logwriter"
	cataloguc "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
	outboxrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/outbox/repository"
	stockrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/stock/repository"
	warehouseuc "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/usecase"
)

// LineInput is a movement line for create APIs.
type LineInput struct {
	ProductID string
	Quantity  int32
	Notes     *string
}

// CreateMovementBase are shared fields for all movement create flows (POST).
// IdempotencyKey is the HTTP Idempotency-Key header (required, §9).
// RequestHashSHA256Hex is SHA-256 hex of the raw JSON body for replay detection.
type CreateMovementBase struct {
	ReferenceNumber      string
	Notes                *string
	Lines                []LineInput
	IdempotencyKey       string
	RequestHashSHA256Hex string
}

// Usecase defines movement application logic.
type Usecase interface {
	Ping() error
	CreateInbound(ctx context.Context, destWarehouseID string, in CreateMovementBase) (movrepo.Movement, error)
	CreateOutbound(ctx context.Context, sourceWarehouseID string, in CreateMovementBase) (movrepo.Movement, error)
	CreateTransfer(ctx context.Context, sourceWarehouseID, destWarehouseID string, in CreateMovementBase) (movrepo.Movement, error)
	CreateAdjustment(ctx context.Context, sourceWarehouseID, destWarehouseID *string, in CreateMovementBase) (movrepo.Movement, error)
	GetMovement(ctx context.Context, movementID string) (movrepo.Movement, error)
	ListMovements(ctx context.Context, in ListMovementsInput) (ListMovementsOutput, error)
	ConfirmMovement(ctx context.Context, movementID string) (movrepo.Movement, error)
	CancelMovement(ctx context.Context, movementID string) (movrepo.Movement, error)
}

// ListMovementsInput filters list.
type ListMovementsInput struct {
	Page    *int
	PerPage *int
	Type    *string
	Status  *string
	Search  *string
	Sort    *string
	Order   *string
}

// ListMovementsOutput is paginated movements (lines omitted in list).
type ListMovementsOutput struct {
	Items   []movrepo.Movement
	Total   int64
	Page    int32
	PerPage int32
}

type usecase struct {
	move     movrepo.Repository
	stock    stockrepo.Repository
	wh       warehouseuc.Usecase
	catalog  cataloguc.Usecase
	auditLog *flowaudit.Writer
	outbox   outboxrepo.Repository
	tx       transaction.Manager
	cache    cachepkg.Cache
}

// New creates movement usecase.
func New(
	move movrepo.Repository,
	stock stockrepo.Repository,
	wh warehouseuc.Usecase,
	catalog cataloguc.Usecase,
	auditLog *flowaudit.Writer,
	outbox outboxrepo.Repository,
	tx transaction.Manager,
	cache cachepkg.Cache,
) Usecase {
	if cache == nil {
		cache = cachepkg.Noop{}
	}
	return &usecase{
		move: move, stock: stock, wh: wh, catalog: catalog,
		auditLog: auditLog, outbox: outbox, tx: tx, cache: cache,
	}
}

func (u *usecase) Ping() error {
	return u.move.Ping()
}

func tenantFromCtx(ctx context.Context) (string, error) {
	return commonjwt.TenantIDFromContext(ctx)
}

func (u *usecase) CreateInbound(ctx context.Context, destWarehouseID string, in CreateMovementBase) (movrepo.Movement, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return movrepo.Movement{}, err
	}
	userID, err := commonjwt.SubjectFromContext(ctx)
	if err != nil {
		return movrepo.Movement{}, err
	}
	dst := strings.TrimSpace(destWarehouseID)
	return u.createDraft(ctx, tid, userID, movrepo.TypeInbound, in, nil, &dst)
}

func (u *usecase) CreateOutbound(ctx context.Context, sourceWarehouseID string, in CreateMovementBase) (movrepo.Movement, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return movrepo.Movement{}, err
	}
	userID, err := commonjwt.SubjectFromContext(ctx)
	if err != nil {
		return movrepo.Movement{}, err
	}
	src := strings.TrimSpace(sourceWarehouseID)
	return u.createDraft(ctx, tid, userID, movrepo.TypeOutbound, in, &src, nil)
}

func (u *usecase) CreateTransfer(ctx context.Context, sourceWarehouseID, destWarehouseID string, in CreateMovementBase) (movrepo.Movement, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return movrepo.Movement{}, err
	}
	userID, err := commonjwt.SubjectFromContext(ctx)
	if err != nil {
		return movrepo.Movement{}, err
	}
	src := strings.TrimSpace(sourceWarehouseID)
	dst := strings.TrimSpace(destWarehouseID)
	return u.createDraft(ctx, tid, userID, movrepo.TypeTransfer, in, &src, &dst)
}

func (u *usecase) CreateAdjustment(ctx context.Context, sourceWarehouseID, destWarehouseID *string, in CreateMovementBase) (movrepo.Movement, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return movrepo.Movement{}, err
	}
	userID, err := commonjwt.SubjectFromContext(ctx)
	if err != nil {
		return movrepo.Movement{}, err
	}
	var src, dst *string
	if sourceWarehouseID != nil && strings.TrimSpace(*sourceWarehouseID) != "" {
		s := strings.TrimSpace(*sourceWarehouseID)
		src = &s
	}
	if destWarehouseID != nil && strings.TrimSpace(*destWarehouseID) != "" {
		d := strings.TrimSpace(*destWarehouseID)
		dst = &d
	}
	return u.createDraft(ctx, tid, userID, movrepo.TypeAdjustment, in, src, dst)
}

func (u *usecase) createDraft(ctx context.Context, tenantID, userID, typ string, in CreateMovementBase, src, dst *string) (movrepo.Movement, error) {
	if err := validateIdempotencyCreate(in); err != nil {
		return movrepo.Movement{}, err
	}
	if err := validateMovementWarehouses(typ, src, dst); err != nil {
		return movrepo.Movement{}, err
	}
	ref := strings.TrimSpace(in.ReferenceNumber)
	if ref == "" {
		return movrepo.Movement{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "reference_number is required"})
	}
	if len(in.Lines) == 0 {
		return movrepo.Movement{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "at least one line is required"})
	}
	ok, err := u.move.UserBelongsToTenant(ctx, tenantID, userID)
	if err != nil {
		return movrepo.Movement{}, err
	}
	if !ok {
		return movrepo.Movement{}, errorcodes.ErrForbidden.WithDetails(map[string]any{"message": "user does not belong to tenant"})
	}

	if err := u.validateWarehousesActive(ctx, src, dst); err != nil {
		return movrepo.Movement{}, err
	}

	for i := range in.Lines {
		l := in.Lines[i]
		if strings.TrimSpace(l.ProductID) == "" || l.Quantity <= 0 {
			return movrepo.Movement{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "each line requires product_id and quantity > 0"})
		}
		if _, err := u.catalog.GetProduct(ctx, strings.TrimSpace(l.ProductID)); err != nil {
			return movrepo.Movement{}, err
		}
	}

	lineIn := make([]movrepo.MovementLineInput, 0, len(in.Lines))
	for i := range in.Lines {
		lineIn = append(lineIn, movrepo.MovementLineInput{
			ProductID: strings.TrimSpace(in.Lines[i].ProductID),
			Quantity:  in.Lines[i].Quantity,
			Notes:     trimNotes(in.Lines[i].Notes),
		})
	}

	mv, replay, err := u.move.Create(ctx, movrepo.CreateMovementInput{
		TenantID:               tenantID,
		Type:                   typ,
		ReferenceNumber:        ref,
		SourceWarehouseID:      src,
		DestinationWarehouseID: dst,
		CreatedBy:              userID,
		Notes:                  trimNotes(in.Notes),
		IdempotencyKey:         strings.TrimSpace(in.IdempotencyKey),
		IdempotencyRequestHash: strings.TrimSpace(strings.ToLower(in.RequestHashSHA256Hex)),
		Lines:                  lineIn,
	})
	if err != nil {
		return movrepo.Movement{}, err
	}
	if u.auditLog != nil && !replay {
		_ = u.auditLog.Log(ctx, flowaudit.Params{
			Action:   "movement.create",
			Entity:   "movement",
			EntityID: mv.ID,
			Before:   nil,
			After:    movementCreateAuditSnapshot(mv),
		})
	}
	return mv, nil
}

func validateIdempotencyCreate(in CreateMovementBase) error {
	key := strings.TrimSpace(in.IdempotencyKey)
	if key == "" {
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "Idempotency-Key header is required"})
	}
	if len(key) > 255 {
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "Idempotency-Key exceeds 255 characters"})
	}
	hash := strings.TrimSpace(strings.ToLower(in.RequestHashSHA256Hex))
	if len(hash) != 64 {
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "invalid request body fingerprint"})
	}
	for _, c := range hash {
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'f' {
			continue
		}
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "invalid request body fingerprint"})
	}
	return nil
}

func trimNotes(p *string) *string {
	return trimString(p)
}

func trimString(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func (u *usecase) validateWarehousesActive(ctx context.Context, src, dst *string) error {
	check := func(id string) error {
		w, err := u.wh.GetWarehouse(ctx, id)
		if err != nil {
			return err
		}
		if !w.IsActive || w.DeletedAt != nil {
			return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "warehouse is inactive or deleted"})
		}
		return nil
	}
	if src != nil {
		if err := check(strings.TrimSpace(*src)); err != nil {
			return err
		}
	}
	if dst != nil {
		if err := check(strings.TrimSpace(*dst)); err != nil {
			return err
		}
	}
	return nil
}

func (u *usecase) GetMovement(ctx context.Context, movementID string) (movrepo.Movement, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return movrepo.Movement{}, err
	}
	return u.move.GetByID(ctx, tid, strings.TrimSpace(movementID))
}

func (u *usecase) ListMovements(ctx context.Context, in ListMovementsInput) (ListMovementsOutput, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return ListMovementsOutput{}, err
	}
	page := 1
	per := 20
	if in.Page != nil {
		page = *in.Page
	}
	if in.PerPage != nil {
		per = *in.PerPage
	}
	pagination.Normalize(&page, &per)

	search := ""
	if in.Search != nil {
		search = strings.TrimSpace(*in.Search)
	}
	sort := "created_at"
	if in.Sort != nil && strings.TrimSpace(*in.Sort) != "" {
		sort = listSortColumn(strings.TrimSpace(*in.Sort))
	}
	order := "DESC"
	if in.Order != nil && strings.EqualFold(strings.TrimSpace(*in.Order), "asc") {
		order = "ASC"
	}

	items, total, err := u.move.List(ctx, tid, movrepo.ListFilter{
		Page: page, PerPage: per,
		Type: in.Type, Status: in.Status,
		Search: search, Sort: sort, Order: order,
	})
	if err != nil {
		return ListMovementsOutput{}, err
	}
	return ListMovementsOutput{
		Items: items, Total: total, Page: int32(page), PerPage: int32(per),
	}, nil
}

func listSortColumn(s string) string {
	switch strings.ToLower(s) {
	case "created_at", "updated_at", "reference_number", "status", "type":
		return s
	default:
		return "created_at"
	}
}

func (u *usecase) CancelMovement(ctx context.Context, movementID string) (movrepo.Movement, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return movrepo.Movement{}, err
	}
	mid := strings.TrimSpace(movementID)
	m, err := u.move.GetByID(ctx, tid, mid)
	if err != nil {
		return movrepo.Movement{}, err
	}
	if m.Status != movrepo.StatusDraft {
		return movrepo.Movement{}, errorcodes.ErrMovementDraft
	}
	before := movementCreateAuditSnapshot(m)
	if err := u.move.UpdateStatus(ctx, tid, mid, movrepo.StatusDraft, movrepo.StatusCancelled); err != nil {
		return movrepo.Movement{}, err
	}
	out, err := u.move.GetByID(ctx, tid, mid)
	if err != nil {
		return movrepo.Movement{}, err
	}
	if u.auditLog != nil {
		_ = u.auditLog.Log(ctx, flowaudit.Params{
			Action:   "movement.cancel",
			Entity:   "movement",
			EntityID: mid,
			Before:   before,
			After:    movementCreateAuditSnapshot(out),
		})
	}
	return out, nil
}

func (u *usecase) ConfirmMovement(ctx context.Context, movementID string) (movrepo.Movement, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return movrepo.Movement{}, err
	}
	mid := strings.TrimSpace(movementID)

	err = u.tx.RunInTx(ctx, func(txCtx context.Context) error {
		mv, err := u.move.GetByIDForUpdate(txCtx, tid, mid)
		if err != nil {
			return err
		}
		if mv.Status != movrepo.StatusDraft {
			return errorcodes.ErrMovementDraft
		}
		if err := u.validateWarehousesActive(txCtx, mv.SourceWarehouseID, mv.DestinationWarehouseID); err != nil {
			return err
		}
		changes, err := applyStock(txCtx, u.stock, tid, mv)
		if err != nil {
			return err
		}
		if err := u.move.UpdateStatus(txCtx, tid, mid, movrepo.StatusDraft, movrepo.StatusConfirmed); err != nil {
			return err
		}
		if err := u.emitAudit(txCtx, mid, mv); err != nil {
			return err
		}
		return u.emitOutbox(txCtx, tid, mid, mv, changes)
	})
	if err != nil {
		return movrepo.Movement{}, err
	}
	_ = u.cache.Delete(ctx, cachepkg.KeyDashboardSummary(tid))
	return u.move.GetByID(ctx, tid, mid)
}
