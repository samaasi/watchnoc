package pagination

import (
	"math"
	"net/http"
	"strconv"
)

const (
	DefaultPage  = 1
	DefaultLimit = 25
	MaxLimit     = 100
)

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

func Parse(r *http.Request) (int, int) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = DefaultPage
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	return page, limit
}

func Meta(page, limit, totalItems int) PaginationMeta {
	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	return PaginationMeta{
		Page:       page,
		PerPage:    limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
