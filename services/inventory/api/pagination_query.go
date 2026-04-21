package api

import (
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

func resolvePagePerPage(c echo.Context, pageParam, perPageParam *int) (int, int) {
	page := 1
	perPage := 10
	if pageParam != nil && *pageParam > 0 {
		page = *pageParam
	}
	if perPageParam != nil && *perPageParam > 0 {
		perPage = *perPageParam
	}

	if pageParam == nil {
		if qp := parsePositiveQueryInt(c, "page"); qp > 0 {
			page = qp
		}
	}
	if perPageParam == nil {
		if qpp := parsePositiveQueryInt(c, "per_page"); qpp > 0 {
			perPage = qpp
		}
	}

	return page, perPage
}

func parsePositiveQueryInt(c echo.Context, key string) int {
	raw := strings.TrimSpace(c.QueryParam(key))
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0
	}
	return n
}
