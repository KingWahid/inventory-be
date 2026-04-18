package api

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	catalogrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
)

func productRepoToStub(p catalogrepo.Product) (stub.Product, error) {
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return stub.Product{}, fmt.Errorf("product id: %w", err)
	}
	tid, err := uuid.Parse(p.TenantID)
	if err != nil {
		return stub.Product{}, fmt.Errorf("tenant id: %w", err)
	}
	var cat *openapi_types.UUID
	if p.CategoryID != nil {
		c, err := uuid.Parse(*p.CategoryID)
		if err != nil {
			return stub.Product{}, fmt.Errorf("category id: %w", err)
		}
		u := openapi_types.UUID(c)
		cat = &u
	}
	price := p.Price
	u := p.Unit
	rl := p.ReorderLevel
	meta := metadataToStubMap(p.Metadata)
	return stub.Product{
		Id:           openapi_types.UUID(id),
		TenantId:     openapi_types.UUID(tid),
		CategoryId:   cat,
		Sku:          p.SKU,
		Name:         p.Name,
		Description:  p.Description,
		Unit:         &u,
		Price:        &price,
		ReorderLevel: &rl,
		Metadata:     meta,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
		DeletedAt:    p.DeletedAt,
	}, nil
}

func metadataToStubMap(raw json.RawMessage) *map[string]interface{} {
	if len(raw) == 0 {
		m := map[string]interface{}{}
		return &m
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		fallback := map[string]interface{}{}
		return &fallback
	}
	return &m
}

func metadataFromStub(m *map[string]interface{}) (json.RawMessage, error) {
	if m == nil {
		return json.RawMessage("{}"), nil
	}
	b, err := json.Marshal(*m)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
