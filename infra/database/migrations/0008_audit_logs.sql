-- +goose Up
CREATE TABLE audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id UUID REFERENCES users(id),
  action VARCHAR(50) NOT NULL,
  entity VARCHAR(100) NOT NULL,
  entity_id UUID NOT NULL,
  before_data JSONB,
  after_data JSONB,
  ip_address VARCHAR(45),
  user_agent TEXT,
  request_id VARCHAR(100),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_tenant ON audit_logs(tenant_id);
CREATE INDEX idx_audit_entity ON audit_logs(tenant_id, entity, entity_id);
CREATE INDEX idx_audit_user ON audit_logs(tenant_id, user_id);
CREATE INDEX idx_audit_action ON audit_logs(tenant_id, action);
CREATE INDEX idx_audit_created ON audit_logs(tenant_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
