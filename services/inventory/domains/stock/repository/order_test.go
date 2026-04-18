package repository

import "testing"

func TestOrderedWarehouseProduct(t *testing.T) {
	t.Parallel()
	a := WarehouseProductKey{WarehouseID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", ProductID: "p1"}
	b := WarehouseProductKey{WarehouseID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", ProductID: "p1"}
	first, second := OrderedWarehouseProduct(a, b)
	if first != a || second != b {
		t.Fatalf("forward order: got first=%+v second=%+v want first=a second=b", first, second)
	}
	first, second = OrderedWarehouseProduct(b, a)
	if first != a || second != b {
		t.Fatalf("reverse order: got first=%+v second=%+v want first=a second=b", first, second)
	}
	if !LessWarehouseProduct(first, second) && !(first.WarehouseID == second.WarehouseID && first.ProductID == second.ProductID) {
		t.Fatal("LessWarehouseProduct(first,second) must hold for ordered pair unless equal")
	}
}

func TestLessWarehouseProductSameWarehouse(t *testing.T) {
	t.Parallel()
	a := WarehouseProductKey{WarehouseID: "wh", ProductID: "aaa"}
	b := WarehouseProductKey{WarehouseID: "wh", ProductID: "zzz"}
	if !LessWarehouseProduct(a, b) {
		t.Fatal("same warehouse: product id order should decide")
	}
	first, second := OrderedWarehouseProduct(a, b)
	if first != a || second != b {
		t.Fatalf("got %+v %+v", first, second)
	}
}
