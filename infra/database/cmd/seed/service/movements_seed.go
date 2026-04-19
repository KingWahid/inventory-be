package service

import (
	"context"
	"database/sql"
	"fmt"
)

// purgeSeedMovements removes prior SEED-* movements (cascade deletes lines). Clears stock for full replay.
func (s *SeedService) purgeSeedMovements(ctx context.Context, tx *sql.Tx, tenantID string) error {
	_, err := tx.ExecContext(ctx,
		`DELETE FROM movements WHERE tenant_id = $1 AND reference_number LIKE $2`,
		tenantID, `SEED-%`,
	)
	if err != nil {
		return fmt.Errorf("delete seed movements: %w", err)
	}
	return nil
}

func (s *SeedService) resetTenantStock(ctx context.Context, tx *sql.Tx, tenantID string) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM stock_balances WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return fmt.Errorf("clear stock_balances: %w", err)
	}
	return nil
}

func (s *SeedService) seedMovementsAndStock(ctx context.Context, tx *sql.Tx, tenantID, createdByUserID string, productBySKU, warehouseByCode map[string]string) error {
	if err := s.purgeSeedMovements(ctx, tx, tenantID); err != nil {
		return err
	}
	if err := s.resetTenantStock(ctx, tx, tenantID); err != nil {
		return err
	}

	for _, mv := range seedMovementScenario {
		movID, err := s.insertMovement(ctx, tx, tenantID, createdByUserID, mv, warehouseByCode)
		if err != nil {
			return err
		}
		if err := s.insertMovementLines(ctx, tx, movID, mv.Lines, productBySKU); err != nil {
			return err
		}
		if mv.Status != "confirmed" {
			continue
		}
		if err := s.applyConfirmedStock(ctx, tx, tenantID, mv, warehouseByCode, productBySKU); err != nil {
			return fmt.Errorf("movement %s: %w", mv.ReferenceNumber, err)
		}
	}
	return nil
}

func (s *SeedService) insertMovement(ctx context.Context, tx *sql.Tx, tenantID, createdBy string, mv MovementSeed, wh map[string]string) (string, error) {
	var srcID, dstID interface{}
	if mv.SourceWHCode != nil {
		id, ok := wh[*mv.SourceWHCode]
		if !ok {
			return "", fmt.Errorf("unknown source warehouse code %q", *mv.SourceWHCode)
		}
		srcID = id
	} else {
		srcID = nil
	}
	if mv.DestWHCode != nil {
		id, ok := wh[*mv.DestWHCode]
		if !ok {
			return "", fmt.Errorf("unknown dest warehouse code %q", *mv.DestWHCode)
		}
		dstID = id
	} else {
		dstID = nil
	}

	var movID string
	err := tx.QueryRowContext(ctx,
		`INSERT INTO movements (
			tenant_id, type, reference_number,
			source_warehouse_id, destination_warehouse_id,
			created_by, status, notes, idempotency_key
		) VALUES (
			$1, $2::movement_type, $3,
			$4, $5,
			$6, $7::movement_status, $8, $9
		) RETURNING id`,
		tenantID, mv.Type, mv.ReferenceNumber,
		srcID, dstID,
		createdBy, mv.Status, nullIfEmpty(mv.Notes), nullIfEmpty(mv.IdempotencyKey),
	).Scan(&movID)
	if err != nil {
		return "", fmt.Errorf("insert movement %s: %w", mv.ReferenceNumber, err)
	}
	return movID, nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (s *SeedService) insertMovementLines(ctx context.Context, tx *sql.Tx, movementID string, lines []MovementLineSeed, productBySKU map[string]string) error {
	for _, ln := range lines {
		pid, ok := productBySKU[ln.SKU]
		if !ok {
			return fmt.Errorf("unknown product SKU %q", ln.SKU)
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO movement_lines (movement_id, product_id, quantity) VALUES ($1, $2, $3)`,
			movementID, pid, ln.Qty,
		)
		if err != nil {
			return fmt.Errorf("insert movement line sku=%s: %w", ln.SKU, err)
		}
	}
	return nil
}

func (s *SeedService) applyConfirmedStock(ctx context.Context, tx *sql.Tx, tenantID string, mv MovementSeed, wh, sku map[string]string) error {
	switch mv.Type {
	case "inbound":
		dst := wh[*mv.DestWHCode]
		for _, ln := range mv.Lines {
			pid := sku[ln.SKU]
			if err := stockDelta(ctx, tx, tenantID, dst, pid, int64(ln.Qty)); err != nil {
				return err
			}
		}
	case "outbound":
		src := wh[*mv.SourceWHCode]
		for _, ln := range mv.Lines {
			pid := sku[ln.SKU]
			if err := stockDelta(ctx, tx, tenantID, src, pid, -int64(ln.Qty)); err != nil {
				return err
			}
		}
	case "transfer":
		src := wh[*mv.SourceWHCode]
		dst := wh[*mv.DestWHCode]
		for _, ln := range mv.Lines {
			pid := sku[ln.SKU]
			q := int64(ln.Qty)
			if err := stockDelta(ctx, tx, tenantID, src, pid, -q); err != nil {
				return err
			}
			if err := stockDelta(ctx, tx, tenantID, dst, pid, q); err != nil {
				return err
			}
		}
	case "adjustment":
		for _, ln := range mv.Lines {
			pid := sku[ln.SKU]
			q := int64(ln.Qty)
			if mv.DestWHCode != nil && mv.SourceWHCode == nil {
				dst := wh[*mv.DestWHCode]
				if err := stockDelta(ctx, tx, tenantID, dst, pid, q); err != nil {
					return err
				}
			} else if mv.SourceWHCode != nil && mv.DestWHCode == nil {
				src := wh[*mv.SourceWHCode]
				if err := stockDelta(ctx, tx, tenantID, src, pid, -q); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("adjustment %s: set exactly one of source or dest warehouse", mv.ReferenceNumber)
			}
		}
	default:
		return fmt.Errorf("unknown movement type %q", mv.Type)
	}
	return nil
}

// stockDelta updates stock_balances (same net effect as domain applyStock).
func stockDelta(ctx context.Context, tx *sql.Tx, tenantID, warehouseID, productID string, delta int64) error {
	var qty int64
	err := tx.QueryRowContext(ctx,
		`SELECT quantity FROM stock_balances
		 WHERE tenant_id = $1 AND warehouse_id = $2 AND product_id = $3
		 FOR UPDATE`,
		tenantID, warehouseID, productID,
	).Scan(&qty)
	if err == sql.ErrNoRows {
		if delta < 0 {
			return fmt.Errorf("no stock row for negative delta (wh=%s prod=%s)", warehouseID, productID)
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO stock_balances (tenant_id, warehouse_id, product_id, quantity, last_movement_at)
			 VALUES ($1, $2, $3, $4, NOW())`,
			tenantID, warehouseID, productID, delta,
		)
		return err
	}
	if err != nil {
		return fmt.Errorf("select stock: %w", err)
	}
	newQty := qty + delta
	if newQty < 0 {
		return fmt.Errorf("stock would go negative (wh=%s prod=%s qty=%d delta=%d)", warehouseID, productID, qty, delta)
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE stock_balances SET quantity = $4, updated_at = NOW(), last_movement_at = NOW()
		 WHERE tenant_id = $1 AND warehouse_id = $2 AND product_id = $3`,
		tenantID, warehouseID, productID, newQty,
	)
	return err
}
