package warehouse

import (
	"go.uber.org/fx"

	cachepkg "github.com/KingWahid/inventory/backend/pkg/cache"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/logwriter"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/handler"
	whrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/repository"
	whuc "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/usecase"
)

// Module wires warehouse domain dependencies.
var Module = fx.Module("warehouse",
	fx.Provide(
		whrepo.New,
		func(repo whrepo.Repository, audit *logwriter.Writer, c cachepkg.Cache) whuc.Usecase {
			return whuc.New(repo, audit, c)
		},
		handler.New,
	),
)
