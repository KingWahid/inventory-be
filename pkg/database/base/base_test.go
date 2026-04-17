package base

import (
	"context"
	"testing"
	"time"
)

func TestNewDefaultTimeout(t *testing.T) {
	repo := New(0)
	ctx, cancel := repo.WithTimeout(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline to be set")
	}
	if time.Until(deadline) > 6*time.Second || time.Until(deadline) < 4*time.Second {
		t.Fatalf("expected ~5s timeout, got %v", time.Until(deadline))
	}
}

func TestNewCustomTimeout(t *testing.T) {
	repo := New(2 * time.Second)
	ctx, cancel := repo.WithTimeout(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline to be set")
	}
	if time.Until(deadline) > 3*time.Second || time.Until(deadline) < 1*time.Second {
		t.Fatalf("expected ~2s timeout, got %v", time.Until(deadline))
	}
}

func TestActiveOnlyClause(t *testing.T) {
	if got := ActiveOnlyClause(""); got != "deleted_at IS NULL" {
		t.Fatalf("unexpected clause without alias: %s", got)
	}
	if got := ActiveOnlyClause("p"); got != "p.deleted_at IS NULL" {
		t.Fatalf("unexpected clause with alias: %s", got)
	}
}
