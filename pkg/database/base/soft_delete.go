package base

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
