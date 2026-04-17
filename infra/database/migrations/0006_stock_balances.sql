-- +goose Up
CREATE TABLE stock_balances (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  warehouse_id UUID NOT NULL REFERENCES warehouses(id),
  product_id UUID NOT NULL REFERENCES products(id),
  quantity INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
  reserved_quantity INT NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0),
  last_movement_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_stock_tenant_warehouse_product UNIQUE (tenant_id, warehouse_id, product_id),
  CONSTRAINT chk_reserved_lte_quantity CHECK (reserved_quantity <= quantity)
);

CREATE INDEX idx_stock_tenant ON stock_balances(tenant_id);
CREATE INDEX idx_stock_warehouse ON stock_balances(warehouse_id);
CREATE INDEX idx_stock_product ON stock_balances(product_id);
CREATE INDEX idx_stock_low ON stock_balances(tenant_id, quantity) WHERE quantity > 0;

-- +goose Down
DROP TABLE IF EXISTS stock_balances;
