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

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want %d got %d body=%s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestRequireAccessJWT_ProtectedWithoutToken(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("test-secret-for-jwt-middleware-testing", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

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

	token, err := jwtSvc.GenerateAccessToken("user-probe", "tenant-probe")
	if err != nil {
		t.Fatal(err)
	}

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
