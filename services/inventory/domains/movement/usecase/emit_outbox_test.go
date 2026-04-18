package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	catalogrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
	cataloguc "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
	outboxrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/outbox/repository"
)

func TestEmitOutbox_propagatesFirstInsertError(t *testing.T) {
	t.Parallel()
	ctx := testClaimsCtx("tenant-1", "user-1")
	u := &usecase{
		outbox: &spyOutbox{failOnCall: 1},
	}
	mv := movrepo.Movement{
		Type:            movrepo.TypeInbound,
		ReferenceNumber: "R1",
		Lines:           []movrepo.MovementLine{{ProductID: "p1", Quantity: 1}},
	}
	err := u.emitOutbox(ctx, "tenant-1", "mov-1", mv, nil)
	if err == nil || err.Error() != "outbox insert failed" {
		t.Fatalf("want outbox error, got %v", err)
	}
}

func TestEmitOutbox_stockBelowThresholdUsesSection10Payload(t *testing.T) {
	t.Parallel()
	ctx := testClaimsCtx("tenant-1", "user-1")
	sp := &spyOutbox{}
	u := &usecase{
		outbox: sp,
		catalog: &catalogStub{
			get: func(ctx context.Context, productID string) (catalogrepo.Product, error) {
				return catalogrepo.Product{
					ID:           productID,
					TenantID:     "tenant-1",
					ReorderLevel: 100,
				}, nil
			},
		},
	}
	mv := movrepo.Movement{
		Type:            movrepo.TypeOutbound,
		ReferenceNumber: "REF",
		Lines:           []movrepo.MovementLine{{ProductID: "prod-a", Quantity: 1}},
	}
	changes := []stockQtyChange{
		{WarehouseID: "wh-1", ProductID: "prod-a", OldQty: 50, NewQty: 40},
	}
	if err := u.emitOutbox(ctx, "tenant-1", "mov-9", mv, changes); err != nil {
		t.Fatal(err)
	}
	var sawBelow bool
	for _, in := range sp.inserts {
		if in.EventType != EventTypeStockBelowThreshold {
			continue
		}
		sawBelow = true
		var p stockBelowThresholdPayload
		if err := json.Unmarshal(in.Payload, &p); err != nil {
			t.Fatal(err)
		}
		if p.TenantID != "tenant-1" || p.WarehouseID != "wh-1" || p.ProductID != "prod-a" ||
			p.CurrentQty != 40 || p.ReorderLevel != 100 {
			t.Fatalf("unexpected payload %+v", p)
		}
	}
	if !sawBelow {
		t.Fatal("expected StockBelowThreshold insert")
	}
}

func TestEmitOutbox_skipsStockBelowWhenReorderDisabledOrAboveThreshold(t *testing.T) {
	t.Parallel()
	ctx := testClaimsCtx("tenant-1", "user-1")
	sp := &spyOutbox{}
	u := &usecase{
		outbox: sp,
		catalog: &catalogStub{
			get: func(ctx context.Context, productID string) (catalogrepo.Product, error) {
				return catalogrepo.Product{
					ID:           productID,
					TenantID:     "tenant-1",
					ReorderLevel: 0,
				}, nil
			},
		},
	}
	mv := movrepo.Movement{ReferenceNumber: "R", Lines: []movrepo.MovementLine{{ProductID: "p", Quantity: 1}}}
	changes := []stockQtyChange{{WarehouseID: "w", ProductID: "p", OldQty: 10, NewQty: 5}}
	if err := u.emitOutbox(ctx, "tenant-1", "m", mv, changes); err != nil {
		t.Fatal(err)
	}
	for _, in := range sp.inserts {
		if in.EventType == EventTypeStockBelowThreshold {
			t.Fatalf("did not expect StockBelowThreshold, got %+v", in)
		}
	}

	sp = &spyOutbox{}
	u = &usecase{
		outbox: sp,
		catalog: &catalogStub{
			get: func(ctx context.Context, productID string) (catalogrepo.Product, error) {
				return catalogrepo.Product{ID: productID, ReorderLevel: 10}, nil
			},
		},
	}
	changes2 := []stockQtyChange{{WarehouseID: "w", ProductID: "p", OldQty: 5, NewQty: 20}}
	if err := u.emitOutbox(ctx, "tenant-1", "m2", mv, changes2); err != nil {
		t.Fatal(err)
	}
	for _, in := range sp.inserts {
		if in.EventType == EventTypeStockBelowThreshold {
			t.Fatalf("did not expect StockBelowThreshold above reorder, got %+v", in)
		}
	}
}

func TestEmitOutbox_catalogLookupFailurePropagates(t *testing.T) {
	t.Parallel()
	ctx := testClaimsCtx("tenant-1", "user-1")
	u := &usecase{
		outbox: &spyOutbox{},
		catalog: &catalogStub{
			get: func(ctx context.Context, productID string) (catalogrepo.Product, error) {
				return catalogrepo.Product{}, errors.New("catalog unavailable")
			},
		},
	}
	mv := movrepo.Movement{ReferenceNumber: "R", Lines: []movrepo.MovementLine{{ProductID: "p", Quantity: 1}}}
	changes := []stockQtyChange{{WarehouseID: "w", ProductID: "p", OldQty: 100, NewQty: 1}}
	err := u.emitOutbox(ctx, "t", "m", mv, changes)
	if err == nil || err.Error() != "catalog unavailable" {
		t.Fatalf("want catalog error, got %v", err)
	}
}

func TestEmitOutbox_eventTypesMatchConstants(t *testing.T) {
	t.Parallel()
	ctx := testClaimsCtx("tenant-1", "user-1")
	sp := &spyOutbox{}
	u := &usecase{outbox: sp}
	mv := movrepo.Movement{
		Type:            movrepo.TypeInbound,
		ReferenceNumber: "R1",
		Lines:           []movrepo.MovementLine{{ProductID: "p1", Quantity: 2}},
	}
	changes := []stockQtyChange{{WarehouseID: "wh-1", ProductID: "p1", OldQty: 0, NewQty: 2}}
	if err := u.emitOutbox(ctx, "tenant-1", "mov-1", mv, changes); err != nil {
		t.Fatal(err)
	}
	if len(sp.inserts) != 2 {
		t.Fatalf("want 2 inserts (MovementCreated + StockChanged), got %d", len(sp.inserts))
	}
	if sp.inserts[0].EventType != EventTypeMovementCreated {
		t.Fatalf("first event want %s got %s", EventTypeMovementCreated, sp.inserts[0].EventType)
	}
	if sp.inserts[1].EventType != EventTypeStockChanged {
		t.Fatalf("second event want %s got %s", EventTypeStockChanged, sp.inserts[1].EventType)
	}
}

// --- test doubles

type spyOutbox struct {
	inserts    []outboxrepo.InsertInput
	failOnCall int
	calls      int
}

func (s *spyOutbox) Ping() error { return nil }

func (s *spyOutbox) Insert(ctx context.Context, in outboxrepo.InsertInput) error {
	s.calls++
	if s.failOnCall != 0 && s.calls == s.failOnCall {
		return errors.New("outbox insert failed")
	}
	s.inserts = append(s.inserts, in)
	return nil
}

func (s *spyOutbox) RelayPublishBatch(context.Context, int, func(outboxrepo.OutboxRow) error) (int, error) {
	return 0, nil
}

type catalogStub struct {
	get func(ctx context.Context, productID string) (catalogrepo.Product, error)
}

func (c *catalogStub) Ping() error { return nil }

func (c *catalogStub) ListCategories(context.Context, cataloguc.ListCategoriesInput) (cataloguc.ListCategoriesOutput, error) {
	return cataloguc.ListCategoriesOutput{}, errors.New("not implemented")
}

func (c *catalogStub) GetCategory(context.Context, string) (catalogrepo.Category, error) {
	return catalogrepo.Category{}, errors.New("not implemented")
}

func (c *catalogStub) CreateCategory(context.Context, cataloguc.CreateCategoryInput) (catalogrepo.Category, error) {
	return catalogrepo.Category{}, errors.New("not implemented")
}

func (c *catalogStub) UpdateCategory(context.Context, string, cataloguc.UpdateCategoryInput) (catalogrepo.Category, error) {
	return catalogrepo.Category{}, errors.New("not implemented")
}

func (c *catalogStub) DeleteCategory(context.Context, string) error {
	return errors.New("not implemented")
}

func (c *catalogStub) ListProducts(context.Context, cataloguc.ListProductsInput) (cataloguc.ListProductsOutput, error) {
	return cataloguc.ListProductsOutput{}, errors.New("not implemented")
}

func (c *catalogStub) GetProduct(ctx context.Context, productID string) (catalogrepo.Product, error) {
	if c.get != nil {
		return c.get(ctx, productID)
	}
	return catalogrepo.Product{ID: productID}, nil
}

func (c *catalogStub) CreateProduct(context.Context, cataloguc.CreateProductInput) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errors.New("not implemented")
}

func (c *catalogStub) UpdateProduct(context.Context, string, cataloguc.UpdateProductInput) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errors.New("not implemented")
}

func (c *catalogStub) DeleteProduct(context.Context, string) error {
	return errors.New("not implemented")
}

func (c *catalogStub) RestoreProduct(context.Context, string) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errors.New("not implemented")
}
