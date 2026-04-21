package usecase

import (
	"context"
	"strings"
	"time"

	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/pkg/common/pagination"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/repository"
)

// ListAuditLogsInput filters list (tenant from JWT).
type ListAuditLogsInput struct {
	Page       *int
	PerPage    *int
	Entity     *string
	EntityID   *string
	Action     *string
	UserID     *string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// ListAuditLogsOutput is paginated audit entries.
type ListAuditLogsOutput struct {
	Items   []repository.Entry
	Total   int64
	Page    int32
	PerPage int32
}

func (u *usecase) ListAuditLogs(ctx context.Context, in ListAuditLogsInput) (ListAuditLogsOutput, error) {
	tid, err := commonjwt.TenantIDFromContext(ctx)
	if err != nil {
		return ListAuditLogsOutput{}, err
	}
	page := 1
	per := 10
	if in.Page != nil {
		page = *in.Page
	}
	if in.PerPage != nil {
		per = *in.PerPage
	}
	pagination.Normalize(&page, &per)

	items, total, err := u.repo.List(ctx, tid, repository.ListFilter{
		Page:        page,
		PerPage:     per,
		Entity:      trimPtr(in.Entity),
		EntityID:    trimPtr(in.EntityID),
		Action:      trimPtr(in.Action),
		UserID:      trimPtr(in.UserID),
		CreatedFrom: in.CreatedFrom,
		CreatedTo:   in.CreatedTo,
	})
	if err != nil {
		return ListAuditLogsOutput{}, err
	}
	return ListAuditLogsOutput{
		Items: items, Total: total, Page: int32(page), PerPage: int32(per),
	}, nil
}

func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}
