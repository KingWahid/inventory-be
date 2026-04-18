package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/labstack/echo/v4"
	goredis "github.com/redis/go-redis/v9"
)

// SSEStock streams Redis Pub/Sub channel stock:{tenant_id} as Server-Sent Events (ARCHITECTURE §11).
// Auth: Authorization: Bearer <access_jwt> or query access_token= (required for browser EventSource).
// Requires Redis (REDIS_ADDR); use HTTPS in production when passing access_token in the query string.
//
// Manual: curl -N -H "Authorization: Bearer <access>" http://localhost:8080/api/v1/inventory/sse/stock
// Then POST confirm movement elsewhere; expect "event: stock_changed" lines with JSON data.
func SSEStock(rdb *goredis.Client, jwtSvc *commonjwt.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		if rdb == nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "SSE requires Redis (set REDIS_ADDR)")
		}
		tok, ok := commonjwt.AccessTokenFromRequest(c.Request())
		if !ok {
			return errorcodes.ErrUnauthorized
		}
		claims, err := jwtSvc.ParseAccess(tok)
		if err != nil {
			return errorcodes.ErrUnauthorized
		}
		if claims.TenantID == "" {
			return errorcodes.ErrUnauthorized
		}

		w := c.Response().Writer
		h := c.Response().Header()
		h.Set(echo.HeaderContentType, "text/event-stream")
		h.Set(echo.HeaderCacheControl, "no-cache")
		h.Set(echo.HeaderConnection, "keep-alive")
		c.Response().WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, "response writer does not support flush")
		}

		ctx := c.Request().Context()
		sub := rdb.Subscribe(ctx, "stock:"+claims.TenantID)
		defer func() { _ = sub.Close() }()

		msgCh := sub.Channel()
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-tick.C:
				_, _ = fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
			case msg, open := <-msgCh:
				if !open {
					return nil
				}
				_, _ = fmt.Fprintf(w, "event: stock_changed\ndata: %s\n\n", msg.Payload)
				flusher.Flush()
			}
		}
	}
}
