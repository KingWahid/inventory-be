package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/your-org/inventory/backend/services/inventory/config"
)

// NewEcho builds Echo with error handling and lifecycle; routes are registered separately (see fx.RegisterRoutes).
func NewEcho(lc fx.Lifecycle, cfg *config.Config, log *zap.Logger) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = httpErrorHandler

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
