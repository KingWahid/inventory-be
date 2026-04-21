package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	auditrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/repository"
	audituc "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/usecase"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
)

type auditListStub struct {
	errOrListSvc
	out audituc.ListAuditLogsOutput
	err error
}

func (a *auditListStub) ListAuditLogs(_ context.Context, in audituc.ListAuditLogsInput) (audituc.ListAuditLogsOutput, error) {
	_ = in
	return a.out, a.err
}

func TestGetApiV1InventoryAuditLogs_OK(t *testing.T) {
	t.Parallel()
	jwtSvc, err := commonjwt.NewService("aud-hdl-jwt-secret-32bytes-min", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tid := uuid.New().String()
	uid := uuid.New().String()
	created := time.Now().UTC().Truncate(time.Second)

	svc := &auditListStub{
		errOrListSvc: errOrListSvc{},
		out: audituc.ListAuditLogsOutput{
			Total:   1,
			Page:    1,
			PerPage: 20,
			Items: []auditrepo.Entry{
				{
					ID:         uuid.New().String(),
					TenantID:   tid,
					UserID:     &uid,
					Action:     "category.create",
					Entity:     "category",
					EntityID:   uuid.New().String(),
					BeforeData: nil,
					AfterData:  []byte(`{}`),
					CreatedAt:  created,
				},
			},
		},
	}

	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(commonjwt.RequireBearerAccessJWT(jwtSvc, InventoryPublicPaths))
	h := NewServerHandler(svc)
	stub.RegisterHandlers(e, h)

	tok, err := jwtSvc.GenerateAccessToken(commonjwt.ClaimsInput{Subject: uid, TenantID: tid})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/audit-logs?page=1&per_page=20", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+tok)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetApiV1InventoryAuditLogs_unauthorized(t *testing.T) {
	t.Parallel()
	jwtSvc, err := commonjwt.NewService("aud-jwt-secret-32bytes-min-x", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(commonjwt.RequireBearerAccessJWT(jwtSvc, InventoryPublicPaths))
	h := NewServerHandler(&auditListStub{errOrListSvc: errOrListSvc{}})
	stub.RegisterHandlers(e, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/audit-logs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("want 401-ish, got %d", rec.Code)
	}
}

func TestAuditEntryToStub_mapsJSON(t *testing.T) {
	t.Parallel()
	id := uuid.New().String()
	e := auditrepo.Entry{
		ID: id, TenantID: uuid.New().String(), Action: "x", Entity: "category", EntityID: uuid.New().String(),
		BeforeData: []byte(`{"a":1}`),
		AfterData:  []byte(`{"b":2}`),
		CreatedAt:  time.Now().UTC(),
	}
	row, err := auditEntryToStub(e)
	if err != nil {
		t.Fatal(err)
	}
	if row.BeforeData == nil || (*row.BeforeData)["a"].(float64) != 1 {
		t.Fatal("before_data mismatch")
	}
	if row.AfterData == nil || (*row.AfterData)["b"].(float64) != 2 {
		t.Fatal("after_data mismatch")
	}
}

func TestAuditEntryToStub_injectsUserNameToAfterData(t *testing.T) {
	t.Parallel()
	name := "Jane Owner"
	e := auditrepo.Entry{
		ID:       uuid.New().String(),
		TenantID: uuid.New().String(),
		Action:   "x",
		Entity:   "category",
		EntityID: uuid.New().String(),
		UserName: &name,
		CreatedAt: time.Now().UTC(),
	}

	row, err := auditEntryToStub(e)
	if err != nil {
		t.Fatal(err)
	}
	if row.AfterData == nil {
		t.Fatal("after_data should not be nil")
	}
	if got := (*row.AfterData)["user_name"]; got != name {
		t.Fatalf("user_name mismatch: got=%v want=%s", got, name)
	}
}
