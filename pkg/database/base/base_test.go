package base

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

func TestNewRepositoryCompatibility(t *testing.T) {
	repo := NewRepository(0)
	ctx, cancel := repo.WithTimeout(context.Background())
	defer cancel()
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("expected deadline to be set")
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

func TestActiveOnlyScope(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed creating sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	query := ActiveOnlyScope("p")(db.Model(struct{}{}))
	if query.Statement == nil || len(query.Statement.Clauses) == 0 {
		t.Fatal("expected scope to add where clause")
	}
}
