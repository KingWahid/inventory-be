package api

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	auditrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/repository"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
)

func auditEntryToStub(e auditrepo.Entry) (stub.AuditLog, error) {
	id, err := uuid.Parse(e.ID)
	if err != nil {
		return stub.AuditLog{}, fmt.Errorf("audit id: %w", err)
	}
	tid, err := uuid.Parse(e.TenantID)
	if err != nil {
		return stub.AuditLog{}, fmt.Errorf("tenant id: %w", err)
	}
	eid, err := uuid.Parse(e.EntityID)
	if err != nil {
		return stub.AuditLog{}, fmt.Errorf("entity id: %w", err)
	}

	var uid *openapi_types.UUID
	if e.UserID != nil && *e.UserID != "" {
		u, err := uuid.Parse(*e.UserID)
		if err != nil {
			return stub.AuditLog{}, fmt.Errorf("user id: %w", err)
		}
		x := openapi_types.UUID(u)
		uid = &x
	}

	before := jsonRawToIfaceMapPtr(e.BeforeData)
	after := jsonRawToIfaceMapPtr(e.AfterData)
	if e.UserName != nil && *e.UserName != "" {
		if after == nil {
			after = &map[string]interface{}{}
		}
		(*after)["user_name"] = *e.UserName
	}

	out := stub.AuditLog{
		Action:     e.Action,
		AfterData:  after,
		BeforeData: before,
		CreatedAt:  e.CreatedAt,
		Entity:     e.Entity,
		EntityId:   openapi_types.UUID(eid),
		Id:         openapi_types.UUID(id),
		IpAddress:  e.IPAddress,
		RequestId:  e.RequestID,
		TenantId:   openapi_types.UUID(tid),
		UserAgent:  e.UserAgent,
		UserId:     uid,
	}
	return out, nil
}

func jsonRawToIfaceMapPtr(b []byte) *map[string]interface{} {
	if len(b) == 0 {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		raw := map[string]interface{}{"_parse_error": err.Error(), "_raw": string(b)}
		return &raw
	}
	return &m
}
