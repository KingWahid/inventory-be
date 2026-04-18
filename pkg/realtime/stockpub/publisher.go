// Package stockpub publishes stock change notifications to Redis Pub/Sub (ARCHITECTURE §11 channel stock:{tenant_id}).
package stockpub

import (
	"context"
	"encoding/json"

	goredis "github.com/redis/go-redis/v9"
)

// Channel returns the Redis Pub/Sub channel name for a tenant.
func Channel(tenantID string) string {
	return "stock:" + tenantID
}

// StockChange is one warehouse/product quantity update after a confirmed movement.
type StockChange struct {
	WarehouseID string `json:"warehouse_id"`
	ProductID   string `json:"product_id"`
	OldQty      int32  `json:"old_qty"`
	NewQty      int32  `json:"new_qty"`
}

// Event is published as JSON and forwarded as SSE event stock_changed data.
type Event struct {
	Event      string        `json:"event"`
	TenantID   string        `json:"tenant_id"`
	MovementID string        `json:"movement_id"`
	Changes    []StockChange `json:"changes"`
}

// Publisher pushes stock_changed messages to Redis (noop when Redis is unavailable).
type Publisher interface {
	PublishStockChanged(ctx context.Context, tenantID, movementID string, changes []StockChange) error
}

// Redis implements Publisher using go-redis PUBLISH.
type Redis struct {
	rdb *goredis.Client
}

// New wraps a Redis client; returns Noop when rdb is nil (dev without REDIS_ADDR).
func New(rdb *goredis.Client) Publisher {
	if rdb == nil {
		return Noop{}
	}
	return &Redis{rdb: rdb}
}

// PublishStockChanged serializes one event and PUBLISHes to stock:{tenantID}.
func (r *Redis) PublishStockChanged(ctx context.Context, tenantID, movementID string, changes []StockChange) error {
	ev := Event{
		Event:      "stock_changed",
		TenantID:   tenantID,
		MovementID: movementID,
		Changes:    changes,
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return r.rdb.Publish(ctx, Channel(tenantID), string(b)).Err()
}

// Noop is used when Redis is disabled; confirm movement still succeeds.
type Noop struct{}

// PublishStockChanged implements Publisher.
func (Noop) PublishStockChanged(context.Context, string, string, []StockChange) error {
	return nil
}
