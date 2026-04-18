package api

import (
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
	"github.com/labstack/echo/v4"
)

// GetApiV1InventoryCategories handles GET /api/v1/inventory/categories.
func (h *ServerHandler) GetApiV1InventoryCategories(c echo.Context, _ stub.GetApiV1InventoryCategoriesParams) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// PostApiV1InventoryCategories handles POST /api/v1/inventory/categories.
func (h *ServerHandler) PostApiV1InventoryCategories(c echo.Context) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// DeleteApiV1InventoryCategoriesCategoryId handles DELETE /api/v1/inventory/categories/{categoryId}.
func (h *ServerHandler) DeleteApiV1InventoryCategoriesCategoryId(c echo.Context, _ stub.CategoryId) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// GetApiV1InventoryCategoriesCategoryId handles GET /api/v1/inventory/categories/{categoryId}.
func (h *ServerHandler) GetApiV1InventoryCategoriesCategoryId(c echo.Context, _ stub.CategoryId) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// PutApiV1InventoryCategoriesCategoryId handles PUT /api/v1/inventory/categories/{categoryId}.
func (h *ServerHandler) PutApiV1InventoryCategoriesCategoryId(c echo.Context, _ stub.CategoryId) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// GetApiV1InventoryProducts handles GET /api/v1/inventory/products.
func (h *ServerHandler) GetApiV1InventoryProducts(c echo.Context, _ stub.GetApiV1InventoryProductsParams) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// PostApiV1InventoryProducts handles POST /api/v1/inventory/products.
func (h *ServerHandler) PostApiV1InventoryProducts(c echo.Context) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// DeleteApiV1InventoryProductsProductId handles DELETE /api/v1/inventory/products/{productId}.
func (h *ServerHandler) DeleteApiV1InventoryProductsProductId(c echo.Context, _ stub.ProductId) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// GetApiV1InventoryProductsProductId handles GET /api/v1/inventory/products/{productId}.
func (h *ServerHandler) GetApiV1InventoryProductsProductId(c echo.Context, _ stub.ProductId) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// PutApiV1InventoryProductsProductId handles PUT /api/v1/inventory/products/{productId}.
func (h *ServerHandler) PutApiV1InventoryProductsProductId(c echo.Context, _ stub.ProductId) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// PostApiV1InventoryProductsProductIdRestore handles POST /api/v1/inventory/products/{productId}/restore.
func (h *ServerHandler) PostApiV1InventoryProductsProductIdRestore(c echo.Context, _ stub.ProductId) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// GetApiV1InventoryWarehouses handles GET /api/v1/inventory/warehouses.
func (h *ServerHandler) GetApiV1InventoryWarehouses(c echo.Context, _ stub.GetApiV1InventoryWarehousesParams) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// PostApiV1InventoryWarehouses handles POST /api/v1/inventory/warehouses.
func (h *ServerHandler) PostApiV1InventoryWarehouses(c echo.Context) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// DeleteApiV1InventoryWarehousesWarehouseId handles DELETE /api/v1/inventory/warehouses/{warehouseId}.
func (h *ServerHandler) DeleteApiV1InventoryWarehousesWarehouseId(c echo.Context, _ stub.WarehouseId) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// GetApiV1InventoryWarehousesWarehouseId handles GET /api/v1/inventory/warehouses/{warehouseId}.
func (h *ServerHandler) GetApiV1InventoryWarehousesWarehouseId(c echo.Context, _ stub.WarehouseId) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// PutApiV1InventoryWarehousesWarehouseId handles PUT /api/v1/inventory/warehouses/{warehouseId}.
func (h *ServerHandler) PutApiV1InventoryWarehousesWarehouseId(c echo.Context, _ stub.WarehouseId) error {
	_ = c
	return errorcodes.ErrNotImplemented
}
