package alertworker

import (
	"context"
	"log"
	"time"

	"github.com/KingWahid/inventory/backend/pkg/eventbus"
)

// Config drives the Redis Streams consumer (ARCHITECTURE §10), e.g. consumer group "alerts" or "notification".
type Config struct {
	ConsumerName string // e.g. worker-1
	Group        string // consumer group name; empty defaults to eventbus.GroupAlerts()
	Block        time.Duration
	Batch        int64
}

func defaultCfg(c Config) Config {
	if c.Group == "" {
		c.Group = eventbus.GroupAlerts()
	}
	if c.ConsumerName == "" {
		c.ConsumerName = "worker-alerts-1"
	}
	if c.Block <= 0 {
		c.Block = 5 * time.Second
	}
	if c.Batch <= 0 {
		c.Batch = 10
	}
	return c
}

// Handler processes a decoded domain event after signature verification.
type Handler func(ctx context.Context, ev eventbus.BaseEvent) error

// Run blocks until ctx done: ensures consumer group, reads via XREADGROUP, verifies HMAC, dispatches and ACKs on success.
func Run(ctx context.Context, bus *eventbus.Client, secret string, h Handler, cfg Config) error {
	cfg = defaultCfg(cfg)
	stream := eventbus.StreamInventoryEvents()
	group := cfg.Group

	if err := bus.EnsureGroup(ctx, stream, group); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msgs, err := bus.ReadGroup(ctx, stream, group, cfg.ConsumerName, cfg.Batch, cfg.Block)
		if err != nil {
			return err
		}
		for _, msg := range msgs {
			ev, err := eventbus.DecodeEvent(msg)
			if err != nil {
				log.Printf("alertworker: decode message %s: %v", msg.ID, err)
				continue
			}
			ok, err := eventbus.VerifyEvent(secret, ev)
			if err != nil || !ok {
				log.Printf("alertworker: verify message %s: ok=%v err=%v", msg.ID, ok, err)
				continue
			}
			if h != nil {
				if err := h(ctx, ev); err != nil {
					log.Printf("alertworker: handler %s: %v", msg.ID, err)
					continue
				}
			}
			if err := bus.Ack(ctx, stream, group, msg.ID); err != nil {
				log.Printf("alertworker: ack %s: %v", msg.ID, err)
			}
		}
	}
}

// StubHandler logs StockBelowThreshold events (replace with email/webhook per §10).
func StubHandler(ctx context.Context, ev eventbus.BaseEvent) error {
	_ = ctx
	if ev.Type != "StockBelowThreshold" {
		return nil
	}
	log.Printf("alertworker: StockBelowThreshold event_id=%s payload=%s", ev.ID, string(ev.Payload))
	return nil
}
