package service

type CategorySeed struct {
	Name        string
	Description string
	SortOrder   int
}

type ProductSeed struct {
	CategoryName string
	SKU          string
	Name         string
	Description  string
	Unit         string
	Price        float64
	ReorderLevel int
}

type WarehouseSeed struct {
	Code     string
	Name     string
	Address  string
	IsActive bool
}

const (
	demoTenantName = "Demo Tenant"
	// demoTenantSlug fixed unique slug for seed (matches tenants.slug UNIQUE).
	demoTenantSlug = "demo-tenant-seed"
	adminEmail     = "admin@demo.local"
	adminPassword = "admin123"
	adminRole     = "owner"
	adminFullName = "Demo Admin"
	defaultReqState = "{}"
)

var demoCategories = []CategorySeed{
	{Name: "Beverages", Description: "Drinks and liquid products", SortOrder: 1},
	{Name: "Snacks", Description: "Packaged snack items", SortOrder: 2},
	{Name: "Household", Description: "Home and cleaning supplies", SortOrder: 3},
}

var demoProducts = []ProductSeed{
	{
		CategoryName: "Beverages",
		SKU:          "BEV-COLA-330",
		Name:         "Cola 330ml",
		Description:  "Carbonated cola drink",
		Unit:         "pcs",
		Price:        8000,
		ReorderLevel: 20,
	},
	{
		CategoryName: "Snacks",
		SKU:          "SNK-POTATO-100",
		Name:         "Potato Chips 100g",
		Description:  "Salted potato chips",
		Unit:         "pcs",
		Price:        12000,
		ReorderLevel: 15,
	},
	{
		CategoryName: "Household",
		SKU:          "HSH-SOAP-500",
		Name:         "Dish Soap 500ml",
		Description:  "Liquid dishwashing soap",
		Unit:         "pcs",
		Price:        18000,
		ReorderLevel: 10,
	},
}

var demoWarehouses = []WarehouseSeed{
	{
		Code:     "WH-JKT-01",
		Name:     "Jakarta Main Warehouse",
		Address:  "Jl. Sudirman No. 1, Jakarta",
		IsActive: true,
	},
	{
		Code:     "WH-BDG-01",
		Name:     "Bandung Backup Warehouse",
		Address:  "Jl. Asia Afrika No. 10, Bandung",
		IsActive: true,
	},
}
