package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestNewRedis_nilIsNoop(t *testing.T) {
	t.Parallel()
	c := NewRedis(nil)
	if _, ok, _ := c.Get(context.Background(), "k"); ok {
		t.Fatal("expected miss for noop")
	}
}

func TestRedis_GetSetDelete(t *testing.T) {
	t.Parallel()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	c := NewRedis(rdb)

	ctx := context.Background()
	key := KeyProduct("tenant-1", "prod-1")
	payload := []byte(`{"sku":"x"}`)
	if err := c.Set(ctx, key, payload, TTLProductOne); err != nil {
		t.Fatal(err)
	}
	got, ok, err := c.Get(ctx, key)
	if err != nil || !ok || string(got) != string(payload) {
		t.Fatalf("got %q ok=%v err=%v", got, ok, err)
	}
	if err := c.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := c.Get(ctx, key); ok {
		t.Fatal("expected miss after delete")
	}
}

func TestRedis_DeletePattern(t *testing.T) {
	t.Parallel()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	c := NewRedis(rdb)
	ctx := context.Background()

	_ = c.Set(ctx, KeyProductsList("t1", "aaa"), []byte(`1`), time.Minute)
	_ = c.Set(ctx, KeyProductsList("t1", "bbb"), []byte(`2`), time.Minute)
	_ = c.Set(ctx, KeyProduct("t1", "id1"), []byte(`3`), time.Minute)

	if err := c.DeletePattern(ctx, PatternProducts("t1")); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := c.Get(ctx, KeyProduct("t1", "id1")); ok {
		t.Fatal("pattern should remove single-product key")
	}
}

func TestQueryFingerprint_stable(t *testing.T) {
	a := ProductsFP(1, 20, "Ab", "name", "asc", "")
	b := ProductsFP(1, 20, "ab", "name", "asc", "")
	if a != b {
		t.Fatalf("search normalize: %s vs %s", a, b)
	}
}
