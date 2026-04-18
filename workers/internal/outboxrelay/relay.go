package outboxrelay

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/KingWahid/inventory/backend/pkg/eventbus"
	outboxrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/outbox/repository"
)

// Config drives polling and backoff for the outbox relay loop.
type Config struct {
	PollInterval time.Duration // sleep when no rows (default 500ms)
	BatchSize    int           // max rows per drain cycle (default 100)
	MinBackoff   time.Duration // initial backoff after publish error (default 200ms)
	MaxBackoff   time.Duration // cap for exponential backoff (default 30s)
}

// DefaultConfig matches ARCHITECTURE §10 relay defaults.
func DefaultConfig() Config {
	return Config{
		PollInterval: 500 * time.Millisecond,
		BatchSize:    100,
		MinBackoff:   200 * time.Millisecond,
		MaxBackoff:   30 * time.Second,
	}
}

// Runner publishes unpublished outbox rows to Redis Streams (inventory.events) using pkg/eventbus.
type Runner struct {
	Repo   outboxrepo.Repository
	Bus    *eventbus.Client
	Secret string // HMAC secret (EVENTBUS_HMAC_SECRET); required for PublishEvent signing
	Config Config
}

// Run blocks until ctx is cancelled. It retries Redis failures with exponential backoff.
func (r *Runner) Run(ctx context.Context) error {
	cfg := r.Config
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 500 * time.Millisecond
	}
	minB := cfg.MinBackoff
	if minB <= 0 {
		minB = 200 * time.Millisecond
	}
	maxB := cfg.MaxBackoff
	if maxB <= 0 {
		maxB = 30 * time.Second
	}
	backoff := minB

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		publishFn := func(row outboxrepo.OutboxRow) error {
			return r.publishOne(ctx, row)
		}

		n, err := r.Repo.RelayPublishBatch(ctx, cfg.BatchSize, publishFn)
		if err != nil {
			sleep := jitterDuration(backoff)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sleep):
			}
			backoff *= 2
			if backoff > maxB {
				backoff = maxB
			}
			continue
		}

		backoff = minB
		if n == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(cfg.PollInterval):
			}
		}
	}
}

func (r *Runner) publishOne(ctx context.Context, row outboxrepo.OutboxRow) error {
	if r.Secret == "" {
		return fmt.Errorf("outboxrelay: EVENTBUS_HMAC_SECRET is required")
	}
	created := row.CreatedAt
	if created.IsZero() {
		created = time.Now().UTC()
	}
	pubAt := time.Now().UTC()
	ev := eventbus.BaseEvent{
		ID:          fmt.Sprintf("outbox:%d", row.ID),
		Type:        row.EventType,
		Version:     1,
		Stream:      eventbus.StreamInventoryEvents(),
		CreatedAt:   created,
		PublishedAt: pubAt,
		Payload:     row.Payload,
	}
	sig, err := eventbus.SignEvent(r.Secret, ev)
	if err != nil {
		return err
	}
	ev.Signature = sig
	_, err = r.Bus.PublishEvent(ctx, ev)
	return err
}

func jitterDuration(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	// Up to 25% jitter
	j := time.Duration(rand.Int63n(int64(d / 4)))
	return d + j
}
