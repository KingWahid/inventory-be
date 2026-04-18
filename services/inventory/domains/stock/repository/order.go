package repository

// WarehouseProductKey identifies one stock_balances row (tenant + warehouse + product; tenant is separate in API).
type WarehouseProductKey struct {
	WarehouseID string
	ProductID   string
}

// LessWarehouseProduct compares (warehouse_id, product_id) lexicographically for deterministic lock ordering (avoids deadlocks on transfer).
func LessWarehouseProduct(a, b WarehouseProductKey) bool {
	if a.WarehouseID != b.WarehouseID {
		return a.WarehouseID < b.WarehouseID
	}
	return a.ProductID < b.ProductID
}

// OrderedWarehouseProduct returns (first, second) such that LessWarehouseProduct(first, second) is true (equal keys allowed — same row).
func OrderedWarehouseProduct(a, b WarehouseProductKey) (WarehouseProductKey, WarehouseProductKey) {
	if LessWarehouseProduct(b, a) {
		return b, a
	}
	return a, b
}
