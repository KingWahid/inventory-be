package outboxrelay

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/KingWahid/inventory/backend/pkg/eventbus"
	outboxrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/outbox/repository"
)

func TestRunner_publishOne_XADDInventoryEvents(t *testing.T) {
	t.Parallel()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	cli, err := eventbus.New(mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	r := &Runner{
		Bus:    cli,
		Secret: "test-hmac-secret-at-least-32-bytes-long-ok",
	}
	row := outboxrepo.OutboxRow{
		ID:        7,
		EventType: "StockChanged",
		Payload:   []byte(`{"tenant_id":"t1"}`),
		CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	if err := r.publishOne(context.Background(), row); err != nil {
		t.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	msgs, err := rdb.XRange(context.Background(), eventbus.StreamInventoryEvents(), "-", "+").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("want 1 stream entry, got %d", len(msgs))
	}
	if msgs[0].Values[eventbus.FieldEventType] != "StockChanged" {
		t.Fatalf("event_type: %v", msgs[0].Values)
	}
	if msgs[0].Values[eventbus.FieldEventID] != "outbox:7" {
		t.Fatalf("event_id: %v", msgs[0].Values)
	}
}

type fakeRepo struct {
	row  outboxrepo.OutboxRow
	done bool
}

func (f *fakeRepo) Ping() error { return nil }

func (f *fakeRepo) Insert(context.Context, outboxrepo.InsertInput) error { return nil }

func (f *fakeRepo) RelayPublishBatch(_ context.Context, _ int, publish func(outboxrepo.OutboxRow) error) (int, error) {
	if f.done {
		return 0, nil
	}
	f.done = true
	if err := publish(f.row); err != nil {
		return 0, err
	}
	return 1, nil
}

func TestRunner_Run_publishesOnce(t *testing.T) {
	t.Parallel()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	cli, err := eventbus.New(mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := &Runner{
		Repo: &fakeRepo{
			row: outboxrepo.OutboxRow{
				ID:        99,
				EventType: "MovementCreated",
				Payload:   []byte(`{}`),
				CreatedAt: time.Now().UTC(),
			},
		},
		Bus:    cli,
		Secret: "test-hmac-secret-at-least-32-bytes-long-ok",
		Config: Config{
			PollInterval: 50 * time.Millisecond,
			BatchSize:    10,
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-errCh

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	n, err := rdb.XLen(context.Background(), eventbus.StreamInventoryEvents()).Result()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want stream length 1, got %d", n)
	}
}
