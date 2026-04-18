package catalog

import (
	"go.uber.org/fx"

	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/logwriter"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/handler"
	catalogrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
	cataloguc "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
)

// Module wires catalog domain dependencies.
var Module = fx.Module("catalog",
	fx.Provide(
		catalogrepo.New,
		func(repo catalogrepo.Repository, audit *logwriter.Writer) cataloguc.Usecase {
			return cataloguc.New(repo, audit)
		},
		handler.New,
	),
)
