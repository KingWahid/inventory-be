package jwt

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccessTokenFromRequest_headerBearer(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/sse/stock", nil)
	r.Header.Set("Authorization", "Bearer abc.def.ghi")
	tok, ok := AccessTokenFromRequest(r)
	if !ok || tok != "abc.def.ghi" {
		t.Fatalf("got %q ok=%v", tok, ok)
	}
}

func TestAccessTokenFromRequest_queryFallback(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/sse/stock?access_token=xyz", nil)
	tok, ok := AccessTokenFromRequest(r)
	if !ok || tok != "xyz" {
		t.Fatalf("got %q ok=%v", tok, ok)
	}
}

func TestAccessTokenFromRequest_headerWinsOverQuery(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/sse/stock?access_token=q", nil)
	r.Header.Set("Authorization", "Bearer from-header")
	tok, ok := AccessTokenFromRequest(r)
	if !ok || tok != "from-header" {
		t.Fatalf("got %q ok=%v", tok, ok)
	}
}

func TestAccessTokenFromRequest_missing(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/sse/stock", nil)
	_, ok := AccessTokenFromRequest(r)
	if ok {
		t.Fatal("expected missing token")
	}
}
