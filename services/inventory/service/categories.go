package service

import (
	"context"

	catalogrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
	cataloguc "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
)

func (s *InventoryService) ListCategories(ctx context.Context, in cataloguc.ListCategoriesInput) (cataloguc.ListCategoriesOutput, error) {
	return s.catalog.ListCategories(ctx, in)
}

func (s *InventoryService) GetCategory(ctx context.Context, categoryID string) (catalogrepo.Category, error) {
	return s.catalog.GetCategory(ctx, categoryID)
}

func (s *InventoryService) CreateCategory(ctx context.Context, in cataloguc.CreateCategoryInput) (catalogrepo.Category, error) {
	return s.catalog.CreateCategory(ctx, in)
}

func (s *InventoryService) UpdateCategory(ctx context.Context, categoryID string, in cataloguc.UpdateCategoryInput) (catalogrepo.Category, error) {
	return s.catalog.UpdateCategory(ctx, categoryID, in)
}

func (s *InventoryService) DeleteCategory(ctx context.Context, categoryID string) error {
	return s.catalog.DeleteCategory(ctx, categoryID)
}
