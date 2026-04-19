package service

import (
	"context"
	"database/sql"
)

// seedAuditSamples inserts a few audit rows for GET /audit-logs smoke tests (no coupling to real entities).
func (s *SeedService) seedAuditSamples(ctx context.Context, tx *sql.Tx, tenantID, userID string) error {
	st := []struct {
		action, entity string
	}{
		{"CREATE", "category"},
		{"UPDATE", "product"},
		{"CONFIRM", "movement"},
	}
	for _, row := range st {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO audit_logs (tenant_id, user_id, action, entity, entity_id, before_data, after_data)
			 VALUES ($1, $2, $3, $4, gen_random_uuid(), '{}'::jsonb, '{"seed":true}'::jsonb)`,
			tenantID, userID, row.action, row.entity,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SeedService) getUserID(ctx context.Context, tx *sql.Tx, tenantID, email string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx,
		`SELECT id FROM users WHERE tenant_id = $1 AND lower(email) = lower($2)`,
		tenantID, email,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *SeedService) loadProductSKUs(ctx context.Context, tx *sql.Tx, tenantID string) (map[string]string, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT sku, id::text FROM products WHERE tenant_id = $1 AND deleted_at IS NULL`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var sku, id string
		if err := rows.Scan(&sku, &id); err != nil {
			return nil, err
		}
		out[sku] = id
	}
	return out, rows.Err()
}

func (s *SeedService) loadWarehouseCodes(ctx context.Context, tx *sql.Tx, tenantID string) (map[string]string, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT code, id::text FROM warehouses WHERE tenant_id = $1`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var code, id string
		if err := rows.Scan(&code, &id); err != nil {
			return nil, err
		}
		out[code] = id
	}
	return out, rows.Err()
}
