package jwt

import (
	"net/http"
	"strings"
)

// AccessTokenFromRequest returns a raw access JWT from (in order):
// 1) Authorization: Bearer <token>
// 2) Query parameter access_token (required for browser EventSource, which cannot set headers).
//
// Production: prefer HTTPS only when using access_token in the query string (token may appear in URLs and logs).
func AccessTokenFromRequest(r *http.Request) (token string, ok bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(h, prefix) {
		t := strings.TrimSpace(strings.TrimPrefix(h, prefix))
		if t != "" {
			return t, true
		}
	}
	if q := strings.TrimSpace(r.URL.Query().Get("access_token")); q != "" {
		return q, true
	}
	return "", false
}
