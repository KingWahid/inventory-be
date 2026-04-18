package audit

import (
	"go.uber.org/fx"

	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/handler"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/logwriter"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/repository"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/usecase"
)

// Module wires audit domain dependencies.
var Module = fx.Module("audit",
	fx.Provide(
		repository.New,
		func(repo repository.Repository) *logwriter.Writer {
			return &logwriter.Writer{Repo: repo}
		},
		usecase.New,
		handler.New,
	),
)
