package base

import "gorm.io/gorm"

const (
	SoftDeleteColumn      = "deleted_at"
	SoftDeleteActiveWhere = "deleted_at IS NULL"
)

func ActiveOnlyClause(alias string) string {
	if alias == "" {
		return SoftDeleteActiveWhere
	}
	return alias + "." + SoftDeleteActiveWhere
}

// ActiveOnlyScope is a reusable GORM scope for soft-delete filtering.
func ActiveOnlyScope(alias string) func(*gorm.DB) *gorm.DB {
	clause := ActiveOnlyClause(alias)
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(clause)
	}
}
