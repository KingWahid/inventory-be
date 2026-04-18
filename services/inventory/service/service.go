package service

import (
	"context"

	catalogrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
	cataloguc "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
)

// Service is the application facade used by HTTP handlers (expand per domain module).
type Service interface {
	PingDB(ctx context.Context) error

	ListCategories(ctx context.Context, in cataloguc.ListCategoriesInput) (cataloguc.ListCategoriesOutput, error)
	GetCategory(ctx context.Context, categoryID string) (catalogrepo.Category, error)
	CreateCategory(ctx context.Context, in cataloguc.CreateCategoryInput) (catalogrepo.Category, error)
	UpdateCategory(ctx context.Context, categoryID string, in cataloguc.UpdateCategoryInput) (catalogrepo.Category, error)
	DeleteCategory(ctx context.Context, categoryID string) error
}
