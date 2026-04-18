-- +goose Up
-- Supports dashboard movement chart / summary queries: confirmed movements by tenant + updated_at range (ARCHITECTURE §9, plan 7.2).
-- Partial index keeps the btree small and matches WHERE status = 'confirmed' filter.
CREATE INDEX idx_movements_tenant_confirmed_updated_at
  ON movements (tenant_id, updated_at DESC)
  WHERE status = 'confirmed'::movement_status;

-- +goose Down
DROP INDEX IF EXISTS idx_movements_tenant_confirmed_updated_at;
