package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/KingWahid/inventory/backend/pkg/eventbus"
)

// StockBelowThresholdPayload mirrors inventory movement emit (ARCHITECTURE §10).
type StockBelowThresholdPayload struct {
	TenantID     string `json:"tenant_id"`
	WarehouseID  string `json:"warehouse_id"`
	ProductID    string `json:"product_id"`
	CurrentQty   int32  `json:"current_qty"`
	ReorderLevel int32  `json:"reorder_level"`
}

// Dispatcher handles verified domain events (stub delivery: structured log + optional webhook).
type Dispatcher struct {
	log        *zap.Logger
	webhookURL string
	httpClient *http.Client
}

// NewDispatcher builds the default dispatcher.
func NewDispatcher(log *zap.Logger, webhookURL string) *Dispatcher {
	return &Dispatcher{
		log:        log,
		webhookURL: webhookURL,
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
}

// Handle implements alertworker.Handler: dispatch notification attempts for supported event types.
func (d *Dispatcher) Handle(ctx context.Context, ev eventbus.BaseEvent) error {
	switch ev.Type {
	case "StockBelowThreshold":
		return d.handleStockBelowThreshold(ctx, ev)
	default:
		return nil
	}
}

func (d *Dispatcher) handleStockBelowThreshold(ctx context.Context, ev eventbus.BaseEvent) error {
	var p StockBelowThresholdPayload
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return fmt.Errorf("dispatch StockBelowThreshold: %w", err)
	}

	d.log.Info("notification_dispatch",
		zap.String("event_id", ev.ID),
		zap.String("event_type", ev.Type),
		zap.String("tenant_id", p.TenantID),
		zap.String("warehouse_id", p.WarehouseID),
		zap.String("product_id", p.ProductID),
		zap.Int32("current_qty", p.CurrentQty),
		zap.Int32("reorder_level", p.ReorderLevel),
	)
	d.log.Info("would_send_email",
		zap.String("event_id", ev.ID),
		zap.String("tenant_id", p.TenantID),
		zap.String("product_id", p.ProductID),
	)

	if d.webhookURL == "" {
		return nil
	}

	body := map[string]any{
		"event_id":      ev.ID,
		"event_type":    ev.Type,
		"tenant_id":     p.TenantID,
		"warehouse_id":  p.WarehouseID,
		"product_id":    p.ProductID,
		"current_qty":   p.CurrentQty,
		"reorder_level": p.ReorderLevel,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		d.log.Warn("notification_webhook_request_failed", zap.Error(err))
		return nil // do not NACK loop forever on transient outbound errors
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		d.log.Warn("notification_webhook_drain_body", zap.Error(err))
	}
	if resp.StatusCode >= 300 {
		d.log.Warn("notification_webhook_bad_status",
			zap.Int("status", resp.StatusCode),
		)
	}
	return nil
}
