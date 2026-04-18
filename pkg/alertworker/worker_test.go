package alertworker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/KingWahid/inventory/backend/pkg/eventbus"
)

func TestRun_verifyAndStubStockBelowThreshold(t *testing.T) {
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

	secret := "test-hmac-secret-at-least-32-bytes-long-ok"
	payload, _ := json.Marshal(map[string]any{"tenant_id": "t1", "product_id": "p1"})
	base := eventbus.BaseEvent{
		ID:          "outbox:1",
		Type:        "StockBelowThreshold",
		Version:     1,
		Stream:      eventbus.StreamInventoryEvents(),
		CreatedAt:   time.Now().UTC(),
		PublishedAt: time.Now().UTC(),
		Payload:     payload,
	}
	sig, err := eventbus.SignEvent(secret, base)
	if err != nil {
		t.Fatal(err)
	}
	base.Signature = sig
	vals, err := base.ToValues()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.Publish(context.Background(), eventbus.StreamInventoryEvents(), vals); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, cli, secret, StubHandler, Config{
			ConsumerName: "test-consumer",
			Block:        100 * time.Millisecond,
			Batch:        1,
		})
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done
}
