package dashboard

import (
	"go.uber.org/fx"

	cachepkg "github.com/KingWahid/inventory/backend/pkg/cache"
	dashrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/dashboard/repository"
	dashuc "github.com/KingWahid/inventory/backend/services/inventory/domains/dashboard/usecase"
)

// Module wires dashboard aggregates + cache-aside for summary.
var Module = fx.Module("dashboard",
	fx.Provide(
		dashrepo.New,
		func(repo dashrepo.Repository, c cachepkg.Cache) dashuc.Usecase {
			return dashuc.New(repo, c)
		},
	),
)
