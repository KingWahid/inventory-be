package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/KingWahid/inventory/backend/services/authentication/service"
	"github.com/labstack/echo/v4"
)

type fakeAuthService struct {
	registerFn func(ctx context.Context, in service.RegisterInput) (service.RegisterResult, error)
	loginFn    func(ctx context.Context, in service.LoginInput) (service.LoginResult, error)
	pingErr    error
}

func (f fakeAuthService) PingDB(context.Context) error { return f.pingErr }

func (f fakeAuthService) RegisterTenantAdmin(ctx context.Context, in service.RegisterInput) (service.RegisterResult, error) {
	return f.registerFn(ctx, in)
}

func (f fakeAuthService) Login(ctx context.Context, in service.LoginInput) (service.LoginResult, error) {
	return f.loginFn(ctx, in)
}

func TestAuthenticationEndpointContract(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler

	h := NewServerHandler(fakeAuthService{
		pingErr: nil,
		registerFn: func(ctx context.Context, in service.RegisterInput) (service.RegisterResult, error) {
			switch in.AdminEmail {
			case "bad@acme.test":
				return service.RegisterResult{}, errorcodes.ErrValidationError
			case "dup@acme.test":
				return service.RegisterResult{}, errorcodes.ErrConflict
			case "panic@acme.test":
				return service.RegisterResult{}, errors.New("unexpected db failure")
			default:
				return service.RegisterResult{
					TenantID: "tenant-contract",
					UserID:   "user-contract",
					Email:    in.AdminEmail,
				}, nil
			}
		},
		loginFn: func(ctx context.Context, in service.LoginInput) (service.LoginResult, error) {
			switch in.Email {
			case "badlogin@acme.test":
				return service.LoginResult{}, errorcodes.ErrValidationError
			case "unauth@acme.test":
				return service.LoginResult{}, errorcodes.ErrUnauthorized
			case "paniclogin@acme.test":
				return service.LoginResult{}, errors.New("unexpected login failure")
			default:
				return service.LoginResult{
					AccessToken:  "token-ok",
					RefreshToken: "refresh-ok",
					TokenType:    "Bearer",
					ExpiresIn:    900,
				}, nil
			}
		},
	})

	e.GET("/health", h.GetHealth)
	e.GET("/ready", h.GetReady)
	e.GET("/api/v1/auth/health", h.GetApiV1AuthHealth)
	e.POST("/api/v1/auth/login", h.PostApiV1AuthLogin)
	e.POST("/api/v1/auth/register", h.PostApiV1AuthRegister)

	assertStatus := func(method, path string, body []byte, want int) {
		t.Helper()
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		if body != nil {
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		t.Logf("%s %s -> got=%d want=%d", method, path, rec.Code, want)
		if rec.Code != want {
			t.Fatalf("%s %s expected %d got %d body=%s", method, path, want, rec.Code, rec.Body.String())
		}
	}

	assertStatus(http.MethodGet, "/health", nil, http.StatusOK)
	assertStatus(http.MethodGet, "/ready", nil, http.StatusOK)
	assertStatus(http.MethodGet, "/api/v1/auth/health", nil, http.StatusOK)

	successBody, _ := json.Marshal(map[string]any{
		"tenant_name": "Acme",
		"admin_name":  "Owner",
		"admin_email": "ok@acme.test",
		"password":    "strongpass123",
	})
	assertStatus(http.MethodPost, "/api/v1/auth/register", successBody, http.StatusCreated)

	badBody, _ := json.Marshal(map[string]any{
		"tenant_name": "Acme",
		"admin_name":  "Owner",
		"admin_email": "bad@acme.test",
		"password":    "strongpass123",
	})
	assertStatus(http.MethodPost, "/api/v1/auth/register", badBody, http.StatusBadRequest)

	dupBody, _ := json.Marshal(map[string]any{
		"tenant_name": "Acme",
		"admin_name":  "Owner",
		"admin_email": "dup@acme.test",
		"password":    "strongpass123",
	})
	assertStatus(http.MethodPost, "/api/v1/auth/register", dupBody, http.StatusConflict)

	errBody, _ := json.Marshal(map[string]any{
		"tenant_name": "Acme",
		"admin_name":  "Owner",
		"admin_email": "panic@acme.test",
		"password":    "strongpass123",
	})
	assertStatus(http.MethodPost, "/api/v1/auth/register", errBody, http.StatusInternalServerError)

	loginSuccess, _ := json.Marshal(map[string]any{
		"email":    "oklogin@acme.test",
		"password": "strongpass123",
	})
	assertStatus(http.MethodPost, "/api/v1/auth/login", loginSuccess, http.StatusOK)

	loginBad, _ := json.Marshal(map[string]any{
		"email":    "badlogin@acme.test",
		"password": "strongpass123",
	})
	assertStatus(http.MethodPost, "/api/v1/auth/login", loginBad, http.StatusBadRequest)

	loginUnauthorized, _ := json.Marshal(map[string]any{
		"email":    "unauth@acme.test",
		"password": "strongpass123",
	})
	assertStatus(http.MethodPost, "/api/v1/auth/login", loginUnauthorized, http.StatusUnauthorized)

	loginInternal, _ := json.Marshal(map[string]any{
		"email":    "paniclogin@acme.test",
		"password": "strongpass123",
	})
	assertStatus(http.MethodPost, "/api/v1/auth/login", loginInternal, http.StatusInternalServerError)
}

func TestPostApiV1AuthRegister_Success(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler

	h := NewServerHandler(fakeAuthService{
		registerFn: func(ctx context.Context, in service.RegisterInput) (service.RegisterResult, error) {
			return service.RegisterResult{
				TenantID: "tenant-1",
				UserID:   "user-1",
				Email:    in.AdminEmail,
			}, nil
		},
		loginFn: func(ctx context.Context, in service.LoginInput) (service.LoginResult, error) {
			return service.LoginResult{}, nil
		},
	})

	body := map[string]any{
		"tenant_name": "Acme",
		"admin_name":  "Owner",
		"admin_email": "owner@acme.test",
		"password":    "strongpass123",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(raw))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.POST("/api/v1/auth/register", h.PostApiV1AuthRegister)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed decode response: %v", err)
	}
	if got["tenant_id"] == "" || got["user_id"] == "" || got["email"] != "owner@acme.test" {
		t.Fatalf("unexpected response body: %+v", got)
	}
}

func TestPostApiV1AuthRegister_Failure(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler

	h := NewServerHandler(fakeAuthService{
		registerFn: func(ctx context.Context, in service.RegisterInput) (service.RegisterResult, error) {
			return service.RegisterResult{}, errorcodes.ErrValidationError
		},
		loginFn: func(ctx context.Context, in service.LoginInput) (service.LoginResult, error) {
			return service.LoginResult{}, nil
		},
	})

	body := map[string]any{
		"tenant_name": "Acme",
		"admin_name":  "Owner",
		"admin_email": "owner@acme.test",
		"password":    "strongpass123",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(raw))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.POST("/api/v1/auth/register", h.PostApiV1AuthRegister)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed decode response: %v", err)
	}
	errObj, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected §9 envelope with error object, got %+v", got)
	}
	if errObj["code"] != errorcodes.CodeValidationError {
		t.Fatalf("expected code %q, got %v", errorcodes.CodeValidationError, errObj["code"])
	}
}

