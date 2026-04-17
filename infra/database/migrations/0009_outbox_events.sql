-- +goose Up
CREATE TABLE outbox_events (
  id BIGSERIAL PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  event_type VARCHAR(100) NOT NULL,
  aggregate_type VARCHAR(100) NOT NULL,
  aggregate_id UUID NOT NULL,
  payload JSONB NOT NULL,
  published BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox_events(created_at)
  WHERE published = false;
CREATE INDEX idx_outbox_tenant ON outbox_events(tenant_id);

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
