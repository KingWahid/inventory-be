package errorcodes

import (
	"errors"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestToHTTP_AppError(t *testing.T) {
	st, ae := ToHTTP(ErrValidationError.WithDetails(map[string]any{"field": "email"}))
	if st != 400 || ae.Code != CodeValidationError {
		t.Fatalf("unexpected %d %+v", st, ae)
	}
}

func TestToHTTP_Unclassified(t *testing.T) {
	st, ae := ToHTTP(errors.New("opaque failure"))
	if st != 500 || ae.Code != CodeInternalError {
		t.Fatalf("want internal got %d %+v", st, ae)
	}
}

func TestToHTTP_JWTParse(t *testing.T) {
	st, ae := ToHTTP(ErrJWTParseToken)
	if st != 401 || ae.Code != CodeUnauthorized {
		t.Fatalf("want 401 unauthorized got %d %+v", st, ae)
	}
}

func TestToHTTP_TenantContextMissing(t *testing.T) {
	st, ae := ToHTTP(ErrTenantContextMissing)
	if st != 401 || ae.Code != CodeUnauthorized {
		t.Fatalf("want 401 unauthorized got %d %+v", st, ae)
	}
}

func TestIsClassified(t *testing.T) {
	if !IsClassified(ErrUnauthorized) {
		t.Fatal("AppError should be classified")
	}
	if IsClassified(errors.New("surprise")) {
		t.Fatal("opaque should not be classified")
	}
	if !IsClassified(echo.NewHTTPError(http.StatusBadRequest, "bad")) {
		t.Fatal("echo HTTPError classified")
	}
}

func TestAPIErrorEnvelope_JSONShape(t *testing.T) {
	env := APIErrorEnvelope{
		Success: false,
		Err: &ErrorPayload{
			Code:    CodeValidationError,
			Message: "bad",
			Details: map[string]any{"x": 1},
		},
		Meta: map[string]interface{}{"request_id": "rid"},
	}
	if env.Success || env.Err == nil || env.Err.Code != CodeValidationError {
		t.Fatalf("%+v", env)
	}
}
