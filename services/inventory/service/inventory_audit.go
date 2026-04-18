package service

import (
	"context"

	audituc "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/usecase"
)

func (s *InventoryService) ListAuditLogs(ctx context.Context, in audituc.ListAuditLogsInput) (audituc.ListAuditLogsOutput, error) {
	return s.audit.ListAuditLogs(ctx, in)
}
