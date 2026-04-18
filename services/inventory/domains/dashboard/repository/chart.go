package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// MovementChartPeriod selects bucket granularity (ARCHITECTURE §9).
type MovementChartPeriod string

const (
	ChartDaily   MovementChartPeriod = "daily"
	ChartWeekly  MovementChartPeriod = "weekly"
	ChartMonthly MovementChartPeriod = "monthly"
)

// MovementChartPoint is one bucket of confirmed movements (UTC).
type MovementChartPoint struct {
	BucketStart   string `json:"bucket_start" gorm:"column:bucket_start"` // YYYY-MM-DD (week/month start)
	MovementCount int64  `json:"movement_count" gorm:"column:movement_count"`
}

// sqlDaily — 30 calendar days UTC ending today; zeros for empty days.
const sqlMovementChartDaily = `
WITH bucket_days AS (
  SELECT gs::date AS bucket_start
  FROM generate_series(
    (timezone('UTC', now()))::date - interval '29 days',
    (timezone('UTC', now()))::date,
    interval '1 day'
  ) AS gs
),
agg AS (
  SELECT ((m.updated_at AT TIME ZONE 'UTC'))::date AS bucket_start, COUNT(*)::bigint AS movement_count
  FROM movements m
  WHERE m.tenant_id = ?::uuid
    AND m.status = 'confirmed'::movement_status
    AND ((m.updated_at AT TIME ZONE 'UTC'))::date >= (timezone('UTC', now()))::date - interval '29 days'
  GROUP BY 1
)
SELECT to_char(b.bucket_start, 'YYYY-MM-DD') AS bucket_start, COALESCE(a.movement_count, 0)::bigint AS movement_count
FROM bucket_days b
LEFT JOIN agg a ON a.bucket_start = b.bucket_start
ORDER BY b.bucket_start`

// sqlWeekly — 12 ISO weeks (Monday) in UTC wall-time ending current week.
const sqlMovementChartWeekly = `
WITH week_buckets AS (
  SELECT gs::date AS bucket_start
  FROM generate_series(
    (date_trunc('week', now() AT TIME ZONE 'UTC')::date - interval '77 days'),
    date_trunc('week', now() AT TIME ZONE 'UTC')::date,
    interval '7 days'
  ) AS gs
),
agg AS (
  SELECT date_trunc('week', m.updated_at AT TIME ZONE 'UTC')::date AS bucket_start,
         COUNT(*)::bigint AS movement_count
  FROM movements m
  WHERE m.tenant_id = ?::uuid
    AND m.status = 'confirmed'::movement_status
    AND date_trunc('week', m.updated_at AT TIME ZONE 'UTC')::date >= date_trunc('week', now() AT TIME ZONE 'UTC')::date - interval '77 days'
  GROUP BY 1
)
SELECT to_char(b.bucket_start, 'YYYY-MM-DD') AS bucket_start, COALESCE(a.movement_count, 0)::bigint AS movement_count
FROM week_buckets b
LEFT JOIN agg a ON a.bucket_start = b.bucket_start
ORDER BY b.bucket_start`

// sqlMonthly — 12 calendar months UTC ending current month.
const sqlMovementChartMonthly = `
WITH month_buckets AS (
  SELECT gs::date AS bucket_start
  FROM generate_series(
    date_trunc('month', now() AT TIME ZONE 'UTC')::date - interval '11 months',
    date_trunc('month', now() AT TIME ZONE 'UTC')::date,
    interval '1 month'
  ) AS gs
),
agg AS (
  SELECT date_trunc('month', m.updated_at AT TIME ZONE 'UTC')::date AS bucket_start,
         COUNT(*)::bigint AS movement_count
  FROM movements m
  WHERE m.tenant_id = ?::uuid
    AND m.status = 'confirmed'::movement_status
    AND date_trunc('month', m.updated_at AT TIME ZONE 'UTC')::date >= date_trunc('month', now() AT TIME ZONE 'UTC')::date - interval '11 months'
  GROUP BY 1
)
SELECT to_char(b.bucket_start, 'YYYY-MM-DD') AS bucket_start, COALESCE(a.movement_count, 0)::bigint AS movement_count
FROM month_buckets b
LEFT JOIN agg a ON a.bucket_start = b.bucket_start
ORDER BY b.bucket_start`

// GetMovementChart returns dense buckets of confirmed movement counts (updated_at UTC).
func (r *repo) GetMovementChart(ctx context.Context, tenantID string, period MovementChartPeriod) ([]MovementChartPoint, error) {
	var sql string
	switch period {
	case ChartDaily:
		sql = sqlMovementChartDaily
	case ChartWeekly:
		sql = sqlMovementChartWeekly
	case ChartMonthly:
		sql = sqlMovementChartMonthly
	default:
		return nil, fmt.Errorf("repository: invalid chart period %q", period)
	}

	var rows []MovementChartPoint
	err := r.db.WithContext(ctx).Raw(sql, tenantID).Scan(&rows).Error
	return rows, err
}

// NormalizeChartPeriod converts request string to MovementChartPeriod.
func NormalizeChartPeriod(s string) (MovementChartPeriod, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", string(ChartDaily):
		return ChartDaily, nil
	case string(ChartWeekly):
		return ChartWeekly, nil
	case string(ChartMonthly):
		return ChartMonthly, nil
	default:
		return "", errors.New("invalid chart period")
	}
}
