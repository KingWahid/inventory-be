package api

import (
	"net/http"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/KingWahid/inventory/backend/pkg/common/httpresponse"
	cataloguc "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
	warehouseuc "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/usecase"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
	"github.com/labstack/echo/v4"
)

// GetApiV1InventoryCategories handles GET /api/v1/inventory/categories.
func (h *ServerHandler) GetApiV1InventoryCategories(c echo.Context, params stub.GetApiV1InventoryCategoriesParams) error {
	ctx := c.Request().Context()
	in := cataloguc.ListCategoriesInput{}
	if params.Page != nil {
		in.Page = params.Page
	}
	if params.PerPage != nil {
		in.PerPage = params.PerPage
	}
	if params.Search != nil {
		s := string(*params.Search)
		in.Search = &s
	}
	if params.Sort != nil {
		s := string(*params.Sort)
		in.Sort = &s
	}
	if params.Order != nil {
		o := string(*params.Order)
		in.Order = &o
	}

	out, err := h.svc.ListCategories(ctx, in)
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	data := make([]stub.Category, 0, len(out.Items))
	for i := range out.Items {
		row, mErr := categoryRepoToStub(out.Items[i])
		if mErr != nil {
			return httpresponse.Fail(c, errorcodes.ErrInternal)
		}
		data = append(data, row)
	}
	pg := httpresponse.PaginationMeta{
		Page:       out.Page,
		PerPage:    out.PerPage,
		Total:      out.Total,
		TotalPages: httpresponse.ComputeTotalPages(out.Total, int64(out.PerPage)),
	}
	return httpresponse.OKList(c, http.StatusOK, data, pg)
}

// PostApiV1InventoryCategories handles POST /api/v1/inventory/categories.
func (h *ServerHandler) PostApiV1InventoryCategories(c echo.Context) error {
	ctx := c.Request().Context()
	var body stub.PostApiV1InventoryCategoriesJSONRequestBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	cat, err := h.svc.CreateCategory(ctx, cataloguc.CreateCategoryInput{
		Name:        body.Name,
		Description: body.Description,
		ParentID:    optionalUUIDString(body.ParentId),
		SortOrder:   body.SortOrder,
	})
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := categoryRepoToStub(cat)
	if err != nil {
		return httpresponse.Fail(c, errorcodes.ErrInternal)
	}
	return httpresponse.OK(c, http.StatusCreated, row)
}

// DeleteApiV1InventoryCategoriesCategoryId handles DELETE /api/v1/inventory/categories/{categoryId}.
func (h *ServerHandler) DeleteApiV1InventoryCategoriesCategoryId(c echo.Context, categoryId stub.CategoryId) error {
	ctx := c.Request().Context()
	if err := h.svc.DeleteCategory(ctx, categoryId.String()); err != nil {
		return httpresponse.Fail(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// GetApiV1InventoryCategoriesCategoryId handles GET /api/v1/inventory/categories/{categoryId}.
func (h *ServerHandler) GetApiV1InventoryCategoriesCategoryId(c echo.Context, categoryId stub.CategoryId) error {
	ctx := c.Request().Context()
	cat, err := h.svc.GetCategory(ctx, categoryId.String())
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := categoryRepoToStub(cat)
	if err != nil {
		return httpresponse.Fail(c, errorcodes.ErrInternal)
	}
	return httpresponse.OK(c, http.StatusOK, row)
}

// PutApiV1InventoryCategoriesCategoryId handles PUT /api/v1/inventory/categories/{categoryId}.
func (h *ServerHandler) PutApiV1InventoryCategoriesCategoryId(c echo.Context, categoryId stub.CategoryId) error {
	ctx := c.Request().Context()
	var body stub.PutApiV1InventoryCategoriesCategoryIdJSONRequestBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	cat, err := h.svc.UpdateCategory(ctx, categoryId.String(), cataloguc.UpdateCategoryInput{
		Name:        body.Name,
		Description: body.Description,
		ParentID:    optionalUUIDString(body.ParentId),
		SortOrder:   body.SortOrder,
	})
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := categoryRepoToStub(cat)
	if err != nil {
		return httpresponse.Fail(c, errorcodes.ErrInternal)
	}
	return httpresponse.OK(c, http.StatusOK, row)
}

// GetApiV1InventoryProducts handles GET /api/v1/inventory/products.
func (h *ServerHandler) GetApiV1InventoryProducts(c echo.Context, params stub.GetApiV1InventoryProductsParams) error {
	ctx := c.Request().Context()
	in := cataloguc.ListProductsInput{}
	if params.Page != nil {
		in.Page = params.Page
	}
	if params.PerPage != nil {
		in.PerPage = params.PerPage
	}
	if params.Search != nil {
		s := string(*params.Search)
		in.Search = &s
	}
	if params.Sort != nil {
		s := string(*params.Sort)
		in.Sort = &s
	}
	if params.Order != nil {
		o := string(*params.Order)
		in.Order = &o
	}
	if params.CategoryId != nil {
		s := params.CategoryId.String()
		in.CategoryID = &s
	}

	out, err := h.svc.ListProducts(ctx, in)
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	data := make([]stub.Product, 0, len(out.Items))
	for i := range out.Items {
		row, mErr := productRepoToStub(out.Items[i])
		if mErr != nil {
			return httpresponse.Fail(c, errorcodes.ErrInternal)
		}
		data = append(data, row)
	}
	pg := httpresponse.PaginationMeta{
		Page:       out.Page,
		PerPage:    out.PerPage,
		Total:      out.Total,
		TotalPages: httpresponse.ComputeTotalPages(out.Total, int64(out.PerPage)),
	}
	return httpresponse.OKList(c, http.StatusOK, data, pg)
}

// PostApiV1InventoryProducts handles POST /api/v1/inventory/products.
func (h *ServerHandler) PostApiV1InventoryProducts(c echo.Context) error {
	ctx := c.Request().Context()
	var body stub.PostApiV1InventoryProductsJSONRequestBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	metaRaw, err := metadataFromStub(body.Metadata)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	p, err := h.svc.CreateProduct(ctx, cataloguc.CreateProductInput{
		CategoryID:   optionalUUIDString(body.CategoryId),
		SKU:          body.Sku,
		Name:         body.Name,
		Description:  body.Description,
		Unit:         body.Unit,
		Price:        body.Price,
		ReorderLevel: body.ReorderLevel,
		MetadataJSON: metaRaw,
	})
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := productRepoToStub(p)
	if err != nil {
		return httpresponse.Fail(c, errorcodes.ErrInternal)
	}
	return httpresponse.OK(c, http.StatusCreated, row)
}

// DeleteApiV1InventoryProductsProductId handles DELETE /api/v1/inventory/products/{productId}.
func (h *ServerHandler) DeleteApiV1InventoryProductsProductId(c echo.Context, productId stub.ProductId) error {
	ctx := c.Request().Context()
	if err := h.svc.DeleteProduct(ctx, productId.String()); err != nil {
		return httpresponse.Fail(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// GetApiV1InventoryProductsProductId handles GET /api/v1/inventory/products/{productId}.
func (h *ServerHandler) GetApiV1InventoryProductsProductId(c echo.Context, productId stub.ProductId) error {
	ctx := c.Request().Context()
	p, err := h.svc.GetProduct(ctx, productId.String())
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := productRepoToStub(p)
	if err != nil {
		return httpresponse.Fail(c, errorcodes.ErrInternal)
	}
	return httpresponse.OK(c, http.StatusOK, row)
}

// PutApiV1InventoryProductsProductId handles PUT /api/v1/inventory/products/{productId}.
func (h *ServerHandler) PutApiV1InventoryProductsProductId(c echo.Context, productId stub.ProductId) error {
	ctx := c.Request().Context()
	var body stub.PutApiV1InventoryProductsProductIdJSONRequestBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	in := cataloguc.UpdateProductInput{
		CategoryID:   optionalUUIDString(body.CategoryId),
		SKU:          body.Sku,
		Name:         body.Name,
		Description:  body.Description,
		Unit:         body.Unit,
		Price:        body.Price,
		ReorderLevel: body.ReorderLevel,
	}
	if body.Metadata != nil {
		metaRaw, err := metadataFromStub(body.Metadata)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		in.MetadataJSON = &metaRaw
	}
	p, err := h.svc.UpdateProduct(ctx, productId.String(), in)
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := productRepoToStub(p)
	if err != nil {
		return httpresponse.Fail(c, errorcodes.ErrInternal)
	}
	return httpresponse.OK(c, http.StatusOK, row)
}

// PostApiV1InventoryProductsProductIdRestore handles POST /api/v1/inventory/products/{productId}/restore.
func (h *ServerHandler) PostApiV1InventoryProductsProductIdRestore(c echo.Context, productId stub.ProductId) error {
	ctx := c.Request().Context()
	p, err := h.svc.RestoreProduct(ctx, productId.String())
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := productRepoToStub(p)
	if err != nil {
		return httpresponse.Fail(c, errorcodes.ErrInternal)
	}
	return httpresponse.OK(c, http.StatusOK, row)
}

// GetApiV1InventoryWarehouses handles GET /api/v1/inventory/warehouses.
func (h *ServerHandler) GetApiV1InventoryWarehouses(c echo.Context, params stub.GetApiV1InventoryWarehousesParams) error {
	ctx := c.Request().Context()
	in := warehouseuc.ListWarehousesInput{}
	if params.Page != nil {
		in.Page = params.Page
	}
	if params.PerPage != nil {
		in.PerPage = params.PerPage
	}
	if params.Search != nil {
		s := string(*params.Search)
		in.Search = &s
	}
	if params.Sort != nil {
		s := string(*params.Sort)
		in.Sort = &s
	}
	if params.Order != nil {
		o := string(*params.Order)
		in.Order = &o
	}

	out, err := h.svc.ListWarehouses(ctx, in)
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	data := make([]stub.Warehouse, 0, len(out.Items))
	for i := range out.Items {
		row, mErr := warehouseRepoToStub(out.Items[i])
		if mErr != nil {
			return httpresponse.Fail(c, errorcodes.ErrInternal)
		}
		data = append(data, row)
	}
	pg := httpresponse.PaginationMeta{
		Page:       out.Page,
		PerPage:    out.PerPage,
		Total:      out.Total,
		TotalPages: httpresponse.ComputeTotalPages(out.Total, int64(out.PerPage)),
	}
	return httpresponse.OKList(c, http.StatusOK, data, pg)
}

// PostApiV1InventoryWarehouses handles POST /api/v1/inventory/warehouses.
func (h *ServerHandler) PostApiV1InventoryWarehouses(c echo.Context) error {
	ctx := c.Request().Context()
	var body stub.PostApiV1InventoryWarehousesJSONRequestBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	w, err := h.svc.CreateWarehouse(ctx, warehouseuc.CreateWarehouseInput{
		Code:     body.Code,
		Name:     body.Name,
		Address:  body.Address,
		IsActive: body.IsActive,
	})
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := warehouseRepoToStub(w)
	if err != nil {
		return httpresponse.Fail(c, errorcodes.ErrInternal)
	}
	return httpresponse.OK(c, http.StatusCreated, row)
}

// DeleteApiV1InventoryWarehousesWarehouseId handles DELETE /api/v1/inventory/warehouses/{warehouseId}.
func (h *ServerHandler) DeleteApiV1InventoryWarehousesWarehouseId(c echo.Context, warehouseId stub.WarehouseId) error {
	ctx := c.Request().Context()
	if err := h.svc.DeleteWarehouse(ctx, warehouseId.String()); err != nil {
		return httpresponse.Fail(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// GetApiV1InventoryWarehousesWarehouseId handles GET /api/v1/inventory/warehouses/{warehouseId}.
func (h *ServerHandler) GetApiV1InventoryWarehousesWarehouseId(c echo.Context, warehouseId stub.WarehouseId) error {
	ctx := c.Request().Context()
	w, err := h.svc.GetWarehouse(ctx, warehouseId.String())
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := warehouseRepoToStub(w)
	if err != nil {
		return httpresponse.Fail(c, errorcodes.ErrInternal)
	}
	return httpresponse.OK(c, http.StatusOK, row)
}

// PutApiV1InventoryWarehousesWarehouseId handles PUT /api/v1/inventory/warehouses/{warehouseId}.
func (h *ServerHandler) PutApiV1InventoryWarehousesWarehouseId(c echo.Context, warehouseId stub.WarehouseId) error {
	ctx := c.Request().Context()
	var body stub.PutApiV1InventoryWarehousesWarehouseIdJSONRequestBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	w, err := h.svc.UpdateWarehouse(ctx, warehouseId.String(), warehouseuc.UpdateWarehouseInput{
		Code:     body.Code,
		Name:     body.Name,
		Address:  body.Address,
		IsActive: body.IsActive,
	})
	if err != nil {
		return httpresponse.Fail(c, err)
	}
	row, err := warehouseRepoToStub(w)
	if err != nil {
		return httpresponse.Fail(c, errorcodes.ErrInternal)
	}
	return httpresponse.OK(c, http.StatusOK, row)
}
