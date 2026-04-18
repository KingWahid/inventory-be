package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OutboxRow is one persisted outbox_events row for relay publishing.
type OutboxRow struct {
	ID            int64
	TenantID      string
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       []byte
	CreatedAt     time.Time
}

// RelayPublishBatch claims up to limit unpublished rows (one PostgreSQL transaction per row).
// For each row it runs: SELECT … FOR UPDATE SKIP LOCKED, publish(row), UPDATE published/published_at.
// If publish returns an error, that row’s transaction rolls back (remains unpublished).
// Parallel relay workers can run safely; SKIP LOCKED hands out disjoint rows.
func (r *repository) RelayPublishBatch(ctx context.Context, limit int, publish func(OutboxRow) error) (int, error) {
	if limit <= 0 {
		limit = 1
	}
	db := r.db.WithContext(ctx)
	n := 0
	for range limit {
		err := db.Transaction(func(tx *gorm.DB) error {
			var row outboxEventRow
			q := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
				Where("published = ?", false).
				Order("id ASC").
				Limit(1)
			if err := q.First(&row).Error; err != nil {
				return err
			}
			out := rowToOutbox(row)
			if err := publish(out); err != nil {
				return err
			}
			now := time.Now().UTC()
			return tx.Model(&outboxEventRow{}).Where("id = ?", row.ID).Updates(map[string]any{
				"published":    true,
				"published_at": now,
			}).Error
		})
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return n, err
		}
		n++
	}
	return n, nil
}

func rowToOutbox(row outboxEventRow) OutboxRow {
	return OutboxRow{
		ID:            row.ID,
		TenantID:      row.TenantID,
		EventType:     row.EventType,
		AggregateType: row.AggregateType,
		AggregateID:   row.AggregateID,
		Payload:       row.Payload,
		CreatedAt:     row.CreatedAt,
	}
}
