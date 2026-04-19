package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/services/authentication/service"
	"github.com/labstack/echo/v4"
)

func TestPostApiV1AuthRefresh_SuccessEnvelope(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler

	h := NewServerHandler(fakeAuthService{
		refreshFn: func(ctx context.Context, in service.RefreshInput) (service.LoginResult, error) {
			if in.RefreshToken != "good-refresh" {
				t.Fatalf("unexpected refresh token %q", in.RefreshToken)
			}
			return service.LoginResult{
				AccessToken:  "new-access",
				RefreshToken: "new-refresh",
				TokenType:    "Bearer",
				ExpiresIn:    86400,
			}, nil
		},
	})

	e.POST("/api/v1/auth/refresh", h.PostApiV1AuthRefresh)

	body := `{"refresh_token":"good-refresh"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
	}
	var env map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env["success"] != true {
		t.Fatalf("want §9 envelope got %+v", env)
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatalf("data: %+v", env)
	}
	if data["access_token"] != "new-access" || data["refresh_token"] != "new-refresh" {
		t.Fatalf("data %+v", data)
	}
}

func TestPostApiV1AuthLogout_NoContent(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("refresh-logout-secret", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(RequireAccessJWT(jwtSvc))

	h := NewServerHandler(fakeAuthService{
		logoutFn: func(ctx context.Context) error {
			cl, ok := commonjwt.ClaimsFromContext(ctx)
			if !ok || cl.Subject != "user-me-1" {
				t.Fatalf("claims: %+v ok=%v", cl, ok)
			}
			return nil
		},
	})

	e.POST("/api/v1/auth/logout", h.PostApiV1AuthLogout)

	tok, err := jwtSvc.GenerateAccessToken(commonjwt.ClaimsInput{
		Subject:  "user-me-1",
		TenantID: "tenant-me-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+tok)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("want 204 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetApiV1AuthMe_SuccessEnvelope(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("me-test-secret", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(RequireAccessJWT(jwtSvc))

	h := NewServerHandler(fakeAuthService{
		meFn: func(ctx context.Context) (service.MeResult, error) {
			return service.MeResult{
				UserID:   "user-me-1",
				TenantID: "tenant-me-1",
				Email:    "me@acme.test",
				FullName: "Me User",
			}, nil
		},
	})

	e.GET("/api/v1/auth/me", h.GetApiV1AuthMe)

	tok, err := jwtSvc.GenerateAccessToken(commonjwt.ClaimsInput{
		Subject:  "user-me-1",
		TenantID: "tenant-me-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+tok)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
	}
	var env map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	data := env["data"].(map[string]any)
	if data["email"] != "me@acme.test" || data["full_name"] != "Me User" {
		t.Fatalf("data %+v", data)
	}
}
