package pagination

// Normalize applies defaults and caps for page (1-based) and limit.
func Normalize(page, limit *int) {
	if page == nil || limit == nil {
		return
	}
	if *page < 1 {
		*page = 1
	}
	if *limit < 1 {
		*limit = 20
	}
	if *limit > 100 {
		*limit = 100
	}
}
