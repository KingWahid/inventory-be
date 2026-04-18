-- +goose Up
-- Raw request hash (SHA-256 hex) for HTTP Idempotency-Key replay detection (§9).
-- Existing rows: NULL (no replay semantics for legacy creates without header hash).
ALTER TABLE movements
  ADD COLUMN idempotency_request_hash VARCHAR(64);

COMMENT ON COLUMN movements.idempotency_request_hash IS 'SHA-256 hex of raw POST body; paired with idempotency_key from Idempotency-Key header';

-- +goose Down
ALTER TABLE movements DROP COLUMN IF EXISTS idempotency_request_hash;
