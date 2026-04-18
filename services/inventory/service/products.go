package service

import (
	"context"

	catalogrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
	cataloguc "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
)

func (s *InventoryService) ListProducts(ctx context.Context, in cataloguc.ListProductsInput) (cataloguc.ListProductsOutput, error) {
	return s.catalog.ListProducts(ctx, in)
}

func (s *InventoryService) GetProduct(ctx context.Context, productID string) (catalogrepo.Product, error) {
	return s.catalog.GetProduct(ctx, productID)
}

func (s *InventoryService) CreateProduct(ctx context.Context, in cataloguc.CreateProductInput) (catalogrepo.Product, error) {
	return s.catalog.CreateProduct(ctx, in)
}

func (s *InventoryService) UpdateProduct(ctx context.Context, productID string, in cataloguc.UpdateProductInput) (catalogrepo.Product, error) {
	return s.catalog.UpdateProduct(ctx, productID, in)
}

func (s *InventoryService) DeleteProduct(ctx context.Context, productID string) error {
	return s.catalog.DeleteProduct(ctx, productID)
}

func (s *InventoryService) RestoreProduct(ctx context.Context, productID string) (catalogrepo.Product, error) {
	return s.catalog.RestoreProduct(ctx, productID)
}
