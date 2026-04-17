package base

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockCache struct {
	getFn func(ctx context.Context, key string) (string, error)
	setFn func(ctx context.Context, key, value string, ttl time.Duration) error
}

func (m mockCache) Get(ctx context.Context, key string) (string, error) {
	if m.getFn == nil {
		return "", nil
	}
	return m.getFn(ctx, key)
}

func (m mockCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if m.setFn == nil {
		return nil
	}
	return m.setFn(ctx, key, value, ttl)
}

type cachedItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestGetFromCacheOrDBCacheHit(t *testing.T) {
	dbCalled := false
	item, err := GetFromCacheOrDB[cachedItem](
		context.Background(),
		mockCache{
			getFn: func(ctx context.Context, key string) (string, error) {
				return `{"id":"p1","name":"cola"}`, nil
			},
		},
		"cache:key",
		5*time.Minute,
		func(ctx context.Context) (cachedItem, error) {
			dbCalled = true
			return cachedItem{}, nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dbCalled {
		t.Fatal("expected dbFn not to be called on cache hit")
	}
	if item.ID != "p1" || item.Name != "cola" {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestGetFromCacheOrDBCacheMiss(t *testing.T) {
	dbCalled := false
	setCalled := false
	item, err := GetFromCacheOrDB[cachedItem](
		context.Background(),
		mockCache{
			getFn: func(ctx context.Context, key string) (string, error) {
				return "", nil
			},
			setFn: func(ctx context.Context, key, value string, ttl time.Duration) error {
				setCalled = true
				return nil
			},
		},
		"cache:key",
		5*time.Minute,
		func(ctx context.Context) (cachedItem, error) {
			dbCalled = true
			return cachedItem{ID: "p2", Name: "chips"}, nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dbCalled {
		t.Fatal("expected dbFn to be called on cache miss")
	}
	if !setCalled {
		t.Fatal("expected cache Set to be called after db fetch")
	}
	if item.ID != "p2" {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestGetFromCacheOrDBFailOpenOnCacheError(t *testing.T) {
	item, err := GetFromCacheOrDB[cachedItem](
		context.Background(),
		mockCache{
			getFn: func(ctx context.Context, key string) (string, error) {
				return "", errors.New("cache down")
			},
		},
		"cache:key",
		5*time.Minute,
		func(ctx context.Context) (cachedItem, error) {
			return cachedItem{ID: "p3", Name: "soap"}, nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ID != "p3" {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestGetFromCacheOrDBReturnsDBError(t *testing.T) {
	expectedErr := errors.New("db failed")
	_, err := GetFromCacheOrDB[cachedItem](
		context.Background(),
		mockCache{},
		"cache:key",
		5*time.Minute,
		func(ctx context.Context) (cachedItem, error) {
			return cachedItem{}, expectedErr
		},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected db error, got: %v", err)
	}
}

func TestGetFromCacheOrDBIntoCacheMiss(t *testing.T) {
	dbCalled := false
	item, err := GetFromCacheOrDBInto[cachedItem](
		context.Background(),
		mockCache{
			getFn: func(ctx context.Context, key string) (string, error) {
				return "", nil
			},
		},
		"cache:key",
		5*time.Minute,
		func(ctx context.Context, out *cachedItem) error {
			dbCalled = true
			out.ID = "p4"
			out.Name = "milk"
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dbCalled {
		t.Fatal("expected queryFn to be called")
	}
	if item.ID != "p4" || item.Name != "milk" {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestGetFromCacheOrDBIntoQueryError(t *testing.T) {
	expectedErr := errors.New("query failed")
	_, err := GetFromCacheOrDBInto[cachedItem](
		context.Background(),
		mockCache{},
		"cache:key",
		5*time.Minute,
		func(ctx context.Context, out *cachedItem) error {
			return expectedErr
		},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected query error, got: %v", err)
	}
}
