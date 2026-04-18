package api

import (
	"fmt"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	catalogrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
)

func categoryRepoToStub(c catalogrepo.Category) (stub.Category, error) {
	id, err := uuid.Parse(c.ID)
	if err != nil {
		return stub.Category{}, fmt.Errorf("category id: %w", err)
	}
	tid, err := uuid.Parse(c.TenantID)
	if err != nil {
		return stub.Category{}, fmt.Errorf("tenant id: %w", err)
	}
	var parent *openapi_types.UUID
	if c.ParentID != nil {
		p, err := uuid.Parse(*c.ParentID)
		if err != nil {
			return stub.Category{}, fmt.Errorf("parent id: %w", err)
		}
		u := openapi_types.UUID(p)
		parent = &u
	}
	so := c.SortOrder
	return stub.Category{
		Id:          openapi_types.UUID(id),
		TenantId:    openapi_types.UUID(tid),
		ParentId:    parent,
		Name:        c.Name,
		Description: c.Description,
		SortOrder:   &so,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		DeletedAt:   c.DeletedAt,
	}, nil
}

func optionalUUIDString(u *openapi_types.UUID) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}
