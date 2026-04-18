package usecase

import (
	"strings"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
)

// Validates movement header warehouses against chk_movement_warehouses (mirror DDL).
func validateMovementWarehouses(movementType string, src, dst *string) error {
	hasSrc := src != nil && strings.TrimSpace(*src) != ""
	hasDst := dst != nil && strings.TrimSpace(*dst) != ""

	switch movementType {
	case movrepo.TypeInbound:
		if hasSrc || !hasDst {
			return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "inbound requires destination warehouse only"})
		}
	case movrepo.TypeOutbound:
		if !hasSrc || hasDst {
			return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "outbound requires source warehouse only"})
		}
	case movrepo.TypeTransfer:
		if !hasSrc || !hasDst {
			return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "transfer requires source and destination warehouses"})
		}
		s, d := strings.TrimSpace(*src), strings.TrimSpace(*dst)
		if s == d {
			return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "transfer source and destination must differ"})
		}
	case movrepo.TypeAdjustment:
		if hasSrc && hasDst {
			return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "adjustment must set exactly one warehouse: source (decrease) OR destination (increase)"})
		}
		if !hasSrc && !hasDst {
			return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "adjustment requires source or destination warehouse"})
		}
	default:
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "unknown movement type"})
	}
	return nil
}
