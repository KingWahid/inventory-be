package logwriter

import (
	"context"
	"encoding/json"

	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/pkg/common/requestmeta"
	auditrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/repository"
)

// Writer appends rows to audit_logs using tenant/user from JWT and HTTP meta from context.
type Writer struct {
	Repo auditrepo.Repository
}

// Params describes one audit event (§14).
type Params struct {
	Action   string
	Entity   string
	EntityID string
	Before   any // encoded as JSON; nil → null in DB
	After    any
}

// Log inserts an audit_logs row. Fails if tenant/user missing from ctx (authenticated routes only).
func (w *Writer) Log(ctx context.Context, p Params) error {
	if w == nil || w.Repo == nil {
		return nil
	}
	tid, err := commonjwt.TenantIDFromContext(ctx)
	if err != nil {
		return err
	}
	uid, err := commonjwt.SubjectFromContext(ctx)
	if err != nil {
		return err
	}
	before, err := marshalSnap(p.Before)
	if err != nil {
		return err
	}
	after, err := marshalSnap(p.After)
	if err != nil {
		return err
	}
	meta := requestmeta.FromContext(ctx)
	return w.Repo.Insert(ctx, auditrepo.InsertInput{
		TenantID:   tid,
		UserID:     &uid,
		Action:     p.Action,
		Entity:     p.Entity,
		EntityID:   p.EntityID,
		BeforeData: before,
		AfterData:  after,
		IPAddress:  meta.IP,
		UserAgent:  meta.UserAgent,
		RequestID:  meta.RequestID,
	})
}

func marshalSnap(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}
