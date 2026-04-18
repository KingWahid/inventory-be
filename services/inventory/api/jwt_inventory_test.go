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

func TestInventoryJWT_PublicPathsNoBearer(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("inv-jwt-test-secret-string", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(commonjwt.RequireBearerAccessJWT(jwtSvc, InventoryPublicPaths))
	e.GET("/health", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d", rec.Code)
	}
}

func TestInventoryJWT_ProtectedRequiresAccessToken(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("inv-jwt-test-secret-string", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(commonjwt.RequireBearerAccessJWT(jwtSvc, InventoryPublicPaths))
	e.GET("/api/v1/inventory/categories", func(c echo.Context) error {
		cl, ok := commonjwt.ClaimsFromContext(c.Request().Context())
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		return c.JSON(http.StatusOK, map[string]string{"sub": cl.Subject})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/categories", nil)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d body=%s", rec.Code, rec.Body.String())
	}

	tok, err := jwtSvc.GenerateAccessToken(commonjwt.ClaimsInput{Subject: "user-x", TenantID: "tenant-x"})
	if err != nil {
		t.Fatal(err)
	}
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/categories", nil)
	req2.Header.Set(echo.HeaderAuthorization, "Bearer "+tok)
	e.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("want 200 got %d", rec2.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(rec2.Body.Bytes(), &body)
	if body["sub"] != "user-x" {
		t.Fatalf("body %+v", body)
	}
}
