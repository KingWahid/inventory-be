package api

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/services/authentication/config"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewEcho builds Echo server and manages lifecycle.
func NewEcho(lc fx.Lifecycle, cfg *config.Config, log *zap.Logger, jwtSvc *commonjwt.Service) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(requestLoggerMiddleware(log))
	e.Use(RequireAccessJWT(jwtSvc))

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

// Throttle successful probe logs so Docker/K8s intervals (~5s) do not flood output.
var (
	probeLogMu       sync.Mutex
	probeLastSuccess time.Time
)

const probeSuccessLogInterval = 30 * time.Minute

func probeLogQuiet(method, rawPath string) bool {
	if method != http.MethodGet {
		return false
	}
	switch rawPath {
	case "/health", "/ready", "/api/v1/auth/health":
		return true
	default:
		return false
	}
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
			if probeLogQuiet(req.Method, req.URL.Path) {
				var shouldLog bool
				probeLogMu.Lock()
				if probeLastSuccess.IsZero() || time.Since(probeLastSuccess) >= probeSuccessLogInterval {
					shouldLog = true
					probeLastSuccess = time.Now()
				}
				probeLogMu.Unlock()
				if shouldLog {
					log.Debug("request completed", fields...)
				}
				return nil
			}
			log.Info("request completed", fields...)
			return nil
		}
	}
}
