package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/labstack/echo/v4"
)

func jwtMiddlewareEcho(jwtSvc *commonjwt.Service) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(RequireAccessJWT(jwtSvc))
	e.GET("/__jwt_probe", func(c echo.Context) error {
		claims, ok := commonjwt.ClaimsFromContext(c.Request().Context())
		if !ok || claims == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "no claims")
		}
		return c.JSON(http.StatusOK, map[string]string{"tenant_id": claims.TenantID, "sub": claims.Subject})
	})
	return e
}

func TestRequireAccessJWT_PublicSkipWithoutToken(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("test-secret-for-jwt-middleware-testing", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(RequireAccessJWT(jwtSvc))

	e.GET("/health", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.POST("/api/v1/auth/refresh", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want %d got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	e.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNoContent {
		t.Fatalf("refresh path should skip JWT want 204 got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestRequireAccessJWT_ProtectedWithoutToken(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("test-secret-for-jwt-middleware-testing", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	e := jwtMiddlewareEcho(jwtSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/__jwt_probe", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want %d got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestRequireAccessJWT_ProtectedWithValidAccessToken(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("test-secret-for-jwt-middleware-testing", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	token, err := jwtSvc.GenerateAccessToken(commonjwt.ClaimsInput{Subject: "user-probe", TenantID: "tenant-probe"})
	if err != nil {
		t.Fatal(err)
	}

	e := jwtMiddlewareEcho(jwtSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/__jwt_probe", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want %d got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["tenant_id"] != "tenant-probe" || got["sub"] != "user-probe" {
		t.Fatalf("unexpected body %+v", got)
	}
}

func TestRequireAccessJWT_RejectsRefreshTokenOnProtectedRoute(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("test-secret-for-jwt-middleware-testing", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	refresh, err := jwtSvc.GenerateRefreshToken(commonjwt.ClaimsInput{Subject: "u", TenantID: "t"})
	if err != nil {
		t.Fatal(err)
	}

	e := jwtMiddlewareEcho(jwtSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/__jwt_probe", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+refresh)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want %d got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestRequireAccessJWT_WrongSigningSecret(t *testing.T) {
	issuer, err := commonjwt.NewService("issuer-secret-middleware-test-xx", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	gate, err := commonjwt.NewService("different-secret-middleware-test-y", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	tok, err := issuer.GenerateAccessToken(commonjwt.ClaimsInput{Subject: "u", TenantID: "t"})
	if err != nil {
		t.Fatal(err)
	}

	e := jwtMiddlewareEcho(gate)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/__jwt_probe", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+tok)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want %d got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestRequireAccessJWT_ExpiredAccessToken(t *testing.T) {
	const secret = "exp-secret-middleware-test-z"
	svc, err := commonjwt.NewService(secret, time.Millisecond, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := svc.GenerateAccessToken(commonjwt.ClaimsInput{Subject: "u", TenantID: "t"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)

	e := jwtMiddlewareEcho(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/__jwt_probe", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+tok)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want %d got %d body=%s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}
