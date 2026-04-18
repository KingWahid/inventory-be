package api

import (
	"net/http"

	"github.com/KingWahid/inventory/backend/services/inventory/stub"
	"github.com/labstack/echo/v4"
)

func inventoryNotImplemented(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, stub.ErrorResponse{Error: "not implemented"})
}

// GetApiV1InventoryCategories handles GET /api/v1/inventory/categories.
func (h *ServerHandler) GetApiV1InventoryCategories(c echo.Context, _ stub.GetApiV1InventoryCategoriesParams) error {
	return inventoryNotImplemented(c)
}

// PostApiV1InventoryCategories handles POST /api/v1/inventory/categories.
func (h *ServerHandler) PostApiV1InventoryCategories(c echo.Context) error {
	return inventoryNotImplemented(c)
}

// DeleteApiV1InventoryCategoriesCategoryId handles DELETE /api/v1/inventory/categories/{categoryId}.
func (h *ServerHandler) DeleteApiV1InventoryCategoriesCategoryId(c echo.Context, _ stub.CategoryId) error {
	return inventoryNotImplemented(c)
}

// GetApiV1InventoryCategoriesCategoryId handles GET /api/v1/inventory/categories/{categoryId}.
func (h *ServerHandler) GetApiV1InventoryCategoriesCategoryId(c echo.Context, _ stub.CategoryId) error {
	return inventoryNotImplemented(c)
}

// PutApiV1InventoryCategoriesCategoryId handles PUT /api/v1/inventory/categories/{categoryId}.
func (h *ServerHandler) PutApiV1InventoryCategoriesCategoryId(c echo.Context, _ stub.CategoryId) error {
	return inventoryNotImplemented(c)
}

// GetApiV1InventoryProducts handles GET /api/v1/inventory/products.
func (h *ServerHandler) GetApiV1InventoryProducts(c echo.Context, _ stub.GetApiV1InventoryProductsParams) error {
	return inventoryNotImplemented(c)
}

// PostApiV1InventoryProducts handles POST /api/v1/inventory/products.
func (h *ServerHandler) PostApiV1InventoryProducts(c echo.Context) error {
	return inventoryNotImplemented(c)
}

// DeleteApiV1InventoryProductsProductId handles DELETE /api/v1/inventory/products/{productId}.
func (h *ServerHandler) DeleteApiV1InventoryProductsProductId(c echo.Context, _ stub.ProductId) error {
	return inventoryNotImplemented(c)
}

// GetApiV1InventoryProductsProductId handles GET /api/v1/inventory/products/{productId}.
func (h *ServerHandler) GetApiV1InventoryProductsProductId(c echo.Context, _ stub.ProductId) error {
	return inventoryNotImplemented(c)
}

// PutApiV1InventoryProductsProductId handles PUT /api/v1/inventory/products/{productId}.
func (h *ServerHandler) PutApiV1InventoryProductsProductId(c echo.Context, _ stub.ProductId) error {
	return inventoryNotImplemented(c)
}

// PostApiV1InventoryProductsProductIdRestore handles POST /api/v1/inventory/products/{productId}/restore.
func (h *ServerHandler) PostApiV1InventoryProductsProductIdRestore(c echo.Context, _ stub.ProductId) error {
	return inventoryNotImplemented(c)
}

// GetApiV1InventoryWarehouses handles GET /api/v1/inventory/warehouses.
func (h *ServerHandler) GetApiV1InventoryWarehouses(c echo.Context, _ stub.GetApiV1InventoryWarehousesParams) error {
	return inventoryNotImplemented(c)
}

// PostApiV1InventoryWarehouses handles POST /api/v1/inventory/warehouses.
func (h *ServerHandler) PostApiV1InventoryWarehouses(c echo.Context) error {
	return inventoryNotImplemented(c)
}

// DeleteApiV1InventoryWarehousesWarehouseId handles DELETE /api/v1/inventory/warehouses/{warehouseId}.
func (h *ServerHandler) DeleteApiV1InventoryWarehousesWarehouseId(c echo.Context, _ stub.WarehouseId) error {
	return inventoryNotImplemented(c)
}

// GetApiV1InventoryWarehousesWarehouseId handles GET /api/v1/inventory/warehouses/{warehouseId}.
func (h *ServerHandler) GetApiV1InventoryWarehousesWarehouseId(c echo.Context, _ stub.WarehouseId) error {
	return inventoryNotImplemented(c)
}

// PutApiV1InventoryWarehousesWarehouseId handles PUT /api/v1/inventory/warehouses/{warehouseId}.
func (h *ServerHandler) PutApiV1InventoryWarehousesWarehouseId(c echo.Context, _ stub.WarehouseId) error {
	return inventoryNotImplemented(c)
}
