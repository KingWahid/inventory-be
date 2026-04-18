package usecase

import (
	"context"
	"errors"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
	stockrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/stock/repository"
)

// stockQtyChange captures before/after quantities for outbox §10 StockChanged events.
type stockQtyChange struct {
	WarehouseID string
	ProductID   string
	OldQty      int32
	NewQty      int32
}

func mapStockNotFound(err error) error {
	if errors.Is(err, errorcodes.ErrNotFound) {
		return errorcodes.ErrInsufficient
	}
	return err
}

func applyStock(ctx context.Context, stock stockrepo.Repository, tenantID string, mv movrepo.Movement) ([]stockQtyChange, error) {
	switch mv.Type {
	case movrepo.TypeInbound:
		return applyInbound(ctx, stock, tenantID, mv)
	case movrepo.TypeOutbound:
		return applyOutbound(ctx, stock, tenantID, mv)
	case movrepo.TypeTransfer:
		return applyTransfer(ctx, stock, tenantID, mv)
	case movrepo.TypeAdjustment:
		return applyAdjustment(ctx, stock, tenantID, mv)
	default:
		return nil, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "unsupported movement type"})
	}
}

func applyInbound(ctx context.Context, stock stockrepo.Repository, tenantID string, mv movrepo.Movement) ([]stockQtyChange, error) {
	dst := *mv.DestinationWarehouseID
	var out []stockQtyChange
	for _, ln := range mv.Lines {
		sb, err := stock.EnsureBalanceRow(ctx, tenantID, dst, ln.ProductID)
		if err != nil {
			return nil, err
		}
		old := sb.Quantity
		if err := stock.ApplyDelta(ctx, tenantID, dst, ln.ProductID, int64(ln.Quantity)); err != nil {
			return nil, err
		}
		out = append(out, stockQtyChange{WarehouseID: dst, ProductID: ln.ProductID, OldQty: old, NewQty: old + ln.Quantity})
	}
	return out, nil
}

func applyOutbound(ctx context.Context, stock stockrepo.Repository, tenantID string, mv movrepo.Movement) ([]stockQtyChange, error) {
	src := *mv.SourceWarehouseID
	var out []stockQtyChange
	for _, ln := range mv.Lines {
		b, err := stock.GetForUpdate(ctx, tenantID, src, ln.ProductID)
		if err != nil {
			return nil, mapStockNotFound(err)
		}
		old := b.Quantity
		if err := stock.ApplyDelta(ctx, tenantID, src, ln.ProductID, -int64(ln.Quantity)); err != nil {
			return nil, err
		}
		out = append(out, stockQtyChange{WarehouseID: src, ProductID: ln.ProductID, OldQty: old, NewQty: old - ln.Quantity})
	}
	return out, nil
}

func applyTransfer(ctx context.Context, stock stockrepo.Repository, tenantID string, mv movrepo.Movement) ([]stockQtyChange, error) {
	src := *mv.SourceWarehouseID
	dst := *mv.DestinationWarehouseID
	var out []stockQtyChange
	for _, ln := range mv.Lines {
		if _, err := stock.EnsureBalanceRow(ctx, tenantID, dst, ln.ProductID); err != nil {
			return nil, err
		}
		if err := stock.TransferDelta(ctx, tenantID, src, dst, ln.ProductID, int64(ln.Quantity)); err != nil {
			return nil, mapStockNotFound(err)
		}
		sbSrc, err := stock.GetForUpdate(ctx, tenantID, src, ln.ProductID)
		if err != nil {
			return nil, mapStockNotFound(err)
		}
		sbDst, err := stock.GetForUpdate(ctx, tenantID, dst, ln.ProductID)
		if err != nil {
			return nil, err
		}
		newS, newD := sbSrc.Quantity, sbDst.Quantity
		oldS := newS + ln.Quantity
		oldD := newD - ln.Quantity
		out = append(out,
			stockQtyChange{WarehouseID: src, ProductID: ln.ProductID, OldQty: oldS, NewQty: newS},
			stockQtyChange{WarehouseID: dst, ProductID: ln.ProductID, OldQty: oldD, NewQty: newD},
		)
	}
	return out, nil
}

func applyAdjustment(ctx context.Context, stock stockrepo.Repository, tenantID string, mv movrepo.Movement) ([]stockQtyChange, error) {
	if mv.DestinationWarehouseID != nil && mv.SourceWarehouseID == nil {
		dst := *mv.DestinationWarehouseID
		var out []stockQtyChange
		for _, ln := range mv.Lines {
			sb, err := stock.EnsureBalanceRow(ctx, tenantID, dst, ln.ProductID)
			if err != nil {
				return nil, err
			}
			old := sb.Quantity
			if err := stock.ApplyDelta(ctx, tenantID, dst, ln.ProductID, int64(ln.Quantity)); err != nil {
				return nil, err
			}
			out = append(out, stockQtyChange{WarehouseID: dst, ProductID: ln.ProductID, OldQty: old, NewQty: old + ln.Quantity})
		}
		return out, nil
	}
	src := *mv.SourceWarehouseID
	var out []stockQtyChange
	for _, ln := range mv.Lines {
		b, err := stock.GetForUpdate(ctx, tenantID, src, ln.ProductID)
		if err != nil {
			return nil, mapStockNotFound(err)
		}
		old := b.Quantity
		if err := stock.ApplyDelta(ctx, tenantID, src, ln.ProductID, -int64(ln.Quantity)); err != nil {
			return nil, mapStockNotFound(err)
		}
		out = append(out, stockQtyChange{WarehouseID: src, ProductID: ln.ProductID, OldQty: old, NewQty: old - ln.Quantity})
	}
	return out, nil
}
