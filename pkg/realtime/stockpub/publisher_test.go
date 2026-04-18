package stockpub

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func TestNew_nilIsNoop(t *testing.T) {
	t.Parallel()
	p := New(nil)
	if err := p.PublishStockChanged(context.Background(), "t1", "m1", nil); err != nil {
		t.Fatal(err)
	}
}

func TestRedis_PublishSubscribeRoundTrip(t *testing.T) {
	t.Parallel()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	sub := rdb.Subscribe(context.Background(), Channel("tenant-a"))
	defer sub.Close()
	ch := sub.Channel()

	p := New(rdb)
	changes := []StockChange{{WarehouseID: "w1", ProductID: "p1", OldQty: 1, NewQty: 2}}
	if err := p.PublishStockChanged(context.Background(), "tenant-a", "mov-1", changes); err != nil {
		t.Fatal(err)
	}

	msg := <-ch
	var got Event
	if err := json.Unmarshal([]byte(msg.Payload), &got); err != nil {
		t.Fatal(err)
	}
	if got.Event != "stock_changed" || got.TenantID != "tenant-a" || got.MovementID != "mov-1" || len(got.Changes) != 1 {
		t.Fatalf("unexpected %+v", got)
	}
}
