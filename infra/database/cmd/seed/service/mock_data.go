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

// UserSeed is upserted per tenant (same dev password via devUserPassword).
type UserSeed struct {
	Email    string
	Role     string
	FullName string
}

// MovementLineSeed references product by SKU after products are seeded.
type MovementLineSeed struct {
	SKU string
	Qty int
}

// MovementSeed defines one movement row + lines. Processed in slice order so stock stays valid.
// Warehouse codes refer to WarehouseSeed.Code; omit side with pointer nil when DB must store NULL (inbound=outbound rules).
type MovementSeed struct {
	ReferenceNumber string
	Type            string // inbound | outbound | transfer | adjustment
	Status          string // draft | confirmed | cancelled
	SourceWHCode    *string
	DestWHCode      *string
	Lines           []MovementLineSeed
	IdempotencyKey  string
	Notes           string
}

const (
	demoTenantName = "Demo Tenant"
	demoTenantSlug = "demo-tenant-seed"

	adminEmail      = "admin@demo.local"
	adminRole       = "owner"
	adminFullName   = "Demo Admin"
	devUserPassword = "admin123" // same for all seeded users (dev only)

	defaultReqState = "{}"
)

var demoCategories = []CategorySeed{
	{Name: "Beverages", Description: "Drinks and liquid products", SortOrder: 1},
	{Name: "Snacks", Description: "Packaged snack items", SortOrder: 2},
	{Name: "Household", Description: "Home and cleaning supplies", SortOrder: 3},
	{Name: "Electronics", Description: "Gadgets and accessories", SortOrder: 4},
	{Name: "Frozen", Description: "Frozen foods", SortOrder: 5},
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
	{
		CategoryName: "Beverages",
		SKU:          "BEV-TEA-500",
		Name:         "Green Tea 500ml",
		Description:  "Unsweetened tea",
		Unit:         "pcs",
		Price:        6500,
		ReorderLevel: 25,
	},
	{
		CategoryName: "Electronics",
		SKU:          "EL-USB-CABLE",
		Name:         "USB-C Cable 1m",
		Description:  "Charging cable",
		Unit:         "pcs",
		Price:        45000,
		ReorderLevel: 30,
	},
	{
		CategoryName: "Frozen",
		SKU:          "FRZ-DUMPLING-250",
		Name:         "Frozen Dumplings 250g",
		Description:  "Pork dumplings",
		Unit:         "pcs",
		Price:        22000,
		ReorderLevel: 12,
	},
	{
		CategoryName: "Snacks",
		SKU:          "SNK-BISCUIT-200",
		Name:         "Butter Biscuits 200g",
		Description:  "Sweet biscuits",
		Unit:         "pcs",
		Price:        9500,
		ReorderLevel: 18,
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
	{
		Code:     "WH-SBY-01",
		Name:     "Surabaya Distribution Hub",
		Address:  "Jl. Tunjungan No. 5, Surabaya",
		IsActive: true,
	},
}

// demoExtraUsers — password devUserPassword (bcrypt applied in seed).
var demoExtraUsers = []UserSeed{
	{Email: "staff@demo.local", Role: "owner", FullName: "Demo Staff"},
	{Email: "viewer@demo.local", Role: "owner", FullName: "Demo Viewer"},
}

// seedMovementScenario — order matters for stock integrity (inbounds before transfer/outbound).
var seedMovementScenario = []MovementSeed{
	{
		ReferenceNumber: "SEED-IN-001",
		Type:            "inbound",
		Status:          "confirmed",
		SourceWHCode:    nil,
		DestWHCode:      strPtr("WH-JKT-01"),
		Lines: []MovementLineSeed{
			{SKU: "BEV-COLA-330", Qty: 500},
			{SKU: "SNK-POTATO-100", Qty: 300},
		},
		IdempotencyKey: "SEED-IDEMP-IN-001",
		Notes:          "Seed bulk inbound Jakarta",
	},
	{
		ReferenceNumber: "SEED-IN-002",
		Type:            "inbound",
		Status:          "confirmed",
		SourceWHCode:    nil,
		DestWHCode:      strPtr("WH-JKT-01"),
		Lines: []MovementLineSeed{
			{SKU: "HSH-SOAP-500", Qty: 150},
			{SKU: "EL-USB-CABLE", Qty: 80},
			{SKU: "SNK-BISCUIT-200", Qty: 60},
			{SKU: "FRZ-DUMPLING-250", Qty: 100},
		},
		IdempotencyKey: "SEED-IDEMP-IN-002",
		Notes:          "Seed inbound Jakarta mixed",
	},
	{
		ReferenceNumber: "SEED-TR-001",
		Type:            "transfer",
		Status:          "confirmed",
		SourceWHCode:    strPtr("WH-JKT-01"),
		DestWHCode:      strPtr("WH-BDG-01"),
		Lines: []MovementLineSeed{
			{SKU: "BEV-COLA-330", Qty: 200},
		},
		IdempotencyKey: "SEED-IDEMP-TR-001",
		Notes:          "Seed transfer JKT to BDG",
	},
	{
		ReferenceNumber: "SEED-OUT-001",
		Type:            "outbound",
		Status:          "confirmed",
		SourceWHCode:    strPtr("WH-BDG-01"),
		DestWHCode:      nil,
		Lines: []MovementLineSeed{
			{SKU: "BEV-COLA-330", Qty: 50},
		},
		IdempotencyKey: "SEED-IDEMP-OUT-001",
		Notes:          "Seed outbound Bandung",
	},
	{
		ReferenceNumber: "SEED-TR-002",
		Type:            "transfer",
		Status:          "confirmed",
		SourceWHCode:    strPtr("WH-JKT-01"),
		DestWHCode:      strPtr("WH-SBY-01"),
		Lines: []MovementLineSeed{
			{SKU: "SNK-POTATO-100", Qty: 80},
			{SKU: "FRZ-DUMPLING-250", Qty: 40},
		},
		IdempotencyKey: "SEED-IDEMP-TR-002",
		Notes:          "Seed transfer JKT to Surabaya",
	},
	{
		ReferenceNumber: "SEED-ADJ-001",
		Type:            "adjustment",
		Status:          "confirmed",
		SourceWHCode:    nil,
		DestWHCode:      strPtr("WH-BDG-01"),
		Lines: []MovementLineSeed{
			{SKU: "HSH-SOAP-500", Qty: 15},
		},
		IdempotencyKey: "SEED-IDEMP-ADJ-001",
		Notes:          "Positive adjustment Bandung",
	},
	{
		ReferenceNumber: "SEED-ADJ-002",
		Type:            "adjustment",
		Status:          "confirmed",
		SourceWHCode:    strPtr("WH-JKT-01"),
		DestWHCode:      nil,
		Lines: []MovementLineSeed{
			{SKU: "SNK-BISCUIT-200", Qty: 5},
		},
		IdempotencyKey: "SEED-IDEMP-ADJ-002",
		Notes:          "Negative adjustment Jakarta (shrinkage)",
	},
	{
		ReferenceNumber: "SEED-DRAFT-001",
		Type:            "inbound",
		Status:          "draft",
		SourceWHCode:    nil,
		DestWHCode:      strPtr("WH-JKT-01"),
		Lines: []MovementLineSeed{
			{SKU: "BEV-TEA-500", Qty: 120},
		},
		IdempotencyKey: "SEED-IDEMP-DRAFT-001",
		Notes:          "Draft inbound (no stock apply)",
	},
	{
		ReferenceNumber: "SEED-CAN-001",
		Type:            "inbound",
		Status:          "cancelled",
		SourceWHCode:    nil,
		DestWHCode:      strPtr("WH-JKT-01"),
		Lines: []MovementLineSeed{
			{SKU: "BEV-TEA-500", Qty: 999},
		},
		IdempotencyKey: "SEED-IDEMP-CAN-001",
		Notes:          "Cancelled inbound",
	},
}

func strPtr(s string) *string { return &s }
