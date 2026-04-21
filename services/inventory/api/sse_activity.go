package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	audituc "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/usecase"
	"github.com/KingWahid/inventory/backend/services/inventory/service"
	"github.com/labstack/echo/v4"
)

type activityChangedPayload struct {
	AuditID   string `json:"audit_id"`
	Entity    string `json:"entity"`
	EntityID  string `json:"entity_id"`
	Action    string `json:"action"`
	CreatedAt string `json:"created_at"`
}

// SSEActivity streams lightweight tenant activity updates as SSE.
// It emits `activity_changed` when newest audit row changes.
func SSEActivity(svc service.Service, jwtSvc *commonjwt.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		tok, ok := commonjwt.AccessTokenFromRequest(c.Request())
		if !ok {
			return errorcodes.ErrUnauthorized
		}
		claims, err := jwtSvc.ParseAccess(tok)
		if err != nil || claims.TenantID == "" {
			return errorcodes.ErrUnauthorized
		}

		ctx := commonjwt.ContextWithClaims(c.Request().Context(), claims)
		w := c.Response().Writer
		h := c.Response().Header()
		h.Set(echo.HeaderContentType, "text/event-stream")
		h.Set(echo.HeaderCacheControl, "no-cache")
		h.Set(echo.HeaderConnection, "keep-alive")
		c.Response().WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, "response writer does not support flush")
		}

		page, perPage := 1, 1
		lastAuditID := ""

		emitLatest := func() error {
			out, listErr := svc.ListAuditLogs(ctx, audituc.ListAuditLogsInput{
				Page:    &page,
				PerPage: &perPage,
			})
			if listErr != nil || len(out.Items) == 0 {
				return listErr
			}
			row := out.Items[0]
			if row.ID == lastAuditID {
				return nil
			}
			lastAuditID = row.ID

			body, marshalErr := json.Marshal(activityChangedPayload{
				AuditID:   row.ID,
				Entity:    row.Entity,
				EntityID:  row.EntityID,
				Action:    row.Action,
				CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
			})
			if marshalErr != nil {
				return marshalErr
			}
			_, _ = fmt.Fprintf(w, "event: activity_changed\ndata: %s\n\n", body)
			flusher.Flush()
			return nil
		}

		_ = emitLatest()
		tick := time.NewTicker(5 * time.Second)
		ping := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		defer ping.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-tick.C:
				_ = emitLatest()
			case <-ping.C:
				_, _ = fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
			}
		}
	}
}
