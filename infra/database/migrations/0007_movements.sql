-- +goose Up
CREATE TYPE movement_type AS ENUM ('inbound', 'outbound', 'transfer', 'adjustment');
CREATE TYPE movement_status AS ENUM ('draft', 'confirmed', 'cancelled');

CREATE TABLE movements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  type movement_type NOT NULL,
  reference_number VARCHAR(100) NOT NULL,
  source_warehouse_id UUID REFERENCES warehouses(id),
  destination_warehouse_id UUID REFERENCES warehouses(id),
  created_by UUID NOT NULL REFERENCES users(id),
  status movement_status NOT NULL DEFAULT 'draft',
  notes TEXT,
  idempotency_key VARCHAR(255),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_movements_tenant_ref UNIQUE (tenant_id, reference_number),
  CONSTRAINT uq_movements_idempotency UNIQUE (tenant_id, idempotency_key),
  CONSTRAINT chk_movement_warehouses CHECK (
    (type = 'inbound' AND source_warehouse_id IS NULL AND destination_warehouse_id IS NOT NULL) OR
    (type = 'outbound' AND source_warehouse_id IS NOT NULL AND destination_warehouse_id IS NULL) OR
    (type = 'transfer' AND source_warehouse_id IS NOT NULL AND destination_warehouse_id IS NOT NULL
      AND source_warehouse_id != destination_warehouse_id) OR
    (type = 'adjustment' AND (source_warehouse_id IS NOT NULL OR destination_warehouse_id IS NOT NULL))
  )
);

CREATE INDEX idx_movements_tenant ON movements(tenant_id);
CREATE INDEX idx_movements_type ON movements(tenant_id, type);
CREATE INDEX idx_movements_status ON movements(tenant_id, status);
CREATE INDEX idx_movements_created ON movements(tenant_id, created_at DESC);
CREATE INDEX idx_movements_source ON movements(source_warehouse_id) WHERE source_warehouse_id IS NOT NULL;
CREATE INDEX idx_movements_dest ON movements(destination_warehouse_id) WHERE destination_warehouse_id IS NOT NULL;

CREATE TABLE movement_lines (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  movement_id UUID NOT NULL REFERENCES movements(id) ON DELETE CASCADE,
  product_id UUID NOT NULL REFERENCES products(id),
  quantity INT NOT NULL CHECK (quantity > 0),
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_movement_lines_movement ON movement_lines(movement_id);
CREATE INDEX idx_movement_lines_product ON movement_lines(product_id);

-- +goose Down
DROP TABLE IF EXISTS movement_lines;
DROP TABLE IF EXISTS movements;
DROP TYPE IF EXISTS movement_status;
DROP TYPE IF EXISTS movement_type;
