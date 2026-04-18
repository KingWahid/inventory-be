package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// Redis key prefixes follow ARCHITECTURE §13. List endpoints use :list:{fp} where fp hashes
// pagination + filters so distinct queries do not share one blob (§13 table shows page-only;
// fingerprint avoids stale rows when per_page/search/sort differ).

const keyPrefix = "cache:t:"

// TTLs from ARCHITECTURE §13.
var (
	TTLProductList      = 5 * time.Minute
	TTLProductOne       = 10 * time.Minute
	TTLCategoryList     = 15 * time.Minute
	TTLWarehouseList    = 15 * time.Minute
	TTLDashboardSummary = 30 * time.Second
	// TTLDashboardChart matches summary; dashboard chart data invalidates on movement confirm (§13 plan 7.2).
	TTLDashboardChart = 30 * time.Second
)

// KeyProduct is cache:t:{tid}:product:{id}
func KeyProduct(tenantID, productID string) string {
	return keyPrefix + tenantID + ":product:" + productID
}

// PatternProducts matches all product-list and single-product cache keys for a tenant.
func PatternProducts(tenantID string) string {
	return keyPrefix + tenantID + ":product*"
}

// KeyProductsList cache:t:{tid}:products:list:{fp}
func KeyProductsList(tenantID, fingerprint string) string {
	return keyPrefix + tenantID + ":products:list:" + fingerprint
}

// KeyCategoriesList cache:t:{tid}:categories:list:{fp}
func KeyCategoriesList(tenantID, fingerprint string) string {
	return keyPrefix + tenantID + ":categories:list:" + fingerprint
}

// PatternCategories matches category list caches for invalidation.
func PatternCategories(tenantID string) string {
	return keyPrefix + tenantID + ":categories:*"
}

// KeyWarehousesList cache:t:{tid}:warehouses:list:{fp}
func KeyWarehousesList(tenantID, fingerprint string) string {
	return keyPrefix + tenantID + ":warehouses:list:" + fingerprint
}

// PatternWarehouses matches warehouse list caches.
func PatternWarehouses(tenantID string) string {
	return keyPrefix + tenantID + ":warehouses:*"
}

// KeyDashboardSummary cache:t:{tid}:dashboard:summary
func KeyDashboardSummary(tenantID string) string {
	return keyPrefix + tenantID + ":dashboard:summary"
}

// KeyDashboardMovementsChart cache:t:{tid}:dashboard:movements:chart:{periodKey}
// periodKey is normalized daily|weekly|monthly (ChartPeriodFingerprint).
func KeyDashboardMovementsChart(tenantID, periodKey string) string {
	return keyPrefix + tenantID + ":dashboard:movements:chart:" + periodKey
}

// PatternDashboardMovementsChart matches all movement chart variants for DELETE after confirm.
func PatternDashboardMovementsChart(tenantID string) string {
	return keyPrefix + tenantID + ":dashboard:movements:chart:*"
}

// ChartPeriodFingerprint normalizes dashboard chart query variant for the cache key (daily|weekly|monthly).
func ChartPeriodFingerprint(period string) string {
	return strings.ToLower(strings.TrimSpace(period))
}

// QueryFingerprint hashes normalized list query dimensions (stable ordering).
func QueryFingerprint(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:])[:16]
}

// CategoriesFP builds fingerprint from normalized category list inputs.
func CategoriesFP(page, perPage int, search, sort, order string) string {
	return QueryFingerprint(strconv.Itoa(page), strconv.Itoa(perPage), strings.ToLower(strings.TrimSpace(search)), sort, order)
}

// ProductsFP builds fingerprint from normalized product list inputs.
func ProductsFP(page, perPage int, search, sort, order, categoryID string) string {
	cat := strings.TrimSpace(categoryID)
	return QueryFingerprint(strconv.Itoa(page), strconv.Itoa(perPage), strings.ToLower(strings.TrimSpace(search)), sort, order, cat)
}

// WarehousesFP builds fingerprint from normalized warehouse list inputs.
func WarehousesFP(page, perPage int, search, sort, order string) string {
	return QueryFingerprint(strconv.Itoa(page), strconv.Itoa(perPage), strings.ToLower(strings.TrimSpace(search)), sort, order)
}
