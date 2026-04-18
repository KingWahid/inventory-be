package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"

	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/pkg/common/requestmeta"
	"github.com/KingWahid/inventory/backend/services/inventory/config"
)

// InventoryPublicPaths skip JWT (exact URL.Path); must align with OpenAPI public probes.
var InventoryPublicPaths = map[string]struct{}{
	"/health":                  {},
	"/ready":                   {},
	"/api/v1/inventory/health": {},
}

// NewEcho builds Echo with error handling and lifecycle; routes are registered separately (see fx.RegisterRoutes).
func NewEcho(lc fx.Lifecycle, cfg *config.Config, log *zap.Logger, jwtSvc *commonjwt.Service) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(requestLoggerMiddleware(log))
	e.Use(commonjwt.RequireBearerAccessJWT(jwtSvc, InventoryPublicPaths))
	e.Use(requestmeta.EchoMiddleware())

	addr := ":" + cfg.AppPort
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Fatal("echo server stopped", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			return e.Shutdown(shutdownCtx)
		},
	})

	return e
}

func requestLoggerMiddleware(log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			req := c.Request()
			res := c.Response()

			fields := []zap.Field{
				zap.String("request_id", res.Header().Get(echo.HeaderXRequestID)),
				zap.String("method", req.Method),
				zap.String("path", c.Path()),
				zap.String("uri", req.RequestURI),
				zap.Int("status", res.Status),
				zap.Int64("bytes_out", res.Size),
				zap.Int64("latency_ms", time.Since(start).Milliseconds()),
			}
			if err != nil {
				fields = append(fields, zap.Error(err))
				log.Warn("request completed with error", fields...)
				return err
			}
			log.Info("request completed", fields...)
			return nil
		}
	}
}
