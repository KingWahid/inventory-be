package httpresponse

// PaginationMeta matches ARCHITECTURE §9 list responses (meta.pagination).
type PaginationMeta struct {
	Page       int32 `json:"page"`
	PerPage    int32 `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int32 `json:"total_pages"`
}

// ComputeTotalPages returns ceiling(total/perPage), or 0 when perPage <= 0 or total <= 0.
func ComputeTotalPages(total, perPage int64) int32 {
	if perPage <= 0 || total <= 0 {
		return 0
	}
	n := (total + perPage - 1) / perPage
	if n > int64(^uint32(0)>>1) {
		return int32(^uint32(0) >> 1)
	}
	return int32(n)
}
