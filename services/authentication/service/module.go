package service

import (
	"time"

	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/services/authentication/config"
	"github.com/KingWahid/inventory/backend/services/authentication/repository"
	"go.uber.org/fx"
)

// Module provides authentication service.
var Module = fx.Module("authentication-service",
	fx.Provide(repository.New),
	fx.Provide(func(cfg *config.Config) (*commonjwt.Service, error) {
		return commonjwt.NewServiceFromSharedOrSplit(
			cfg.JWTSecret,
			cfg.JWTAccessSecret,
			cfg.JWTRefreshSecret,
			time.Duration(cfg.JWTAccessTTLSeconds)*time.Second,
			time.Duration(cfg.JWTRefreshTTLSeconds)*time.Second,
			cfg.JWTIssuer,
			cfg.JWTAudience,
		)
	}),
	fx.Provide(func(repo repository.Repository, jwtSvc *commonjwt.Service, cfg *config.Config) Service {
		return NewAuthenticationService(repo, jwtSvc, int64(cfg.JWTAccessTTLSeconds))
	}),
)
