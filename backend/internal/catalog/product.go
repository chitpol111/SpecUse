package catalog

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("product not found")
)

// Product exposes only the six fields already agreed with the frontend.
// TODO(FE/BE): agree on compatibility JSON fields before exposing typed details.
type Product struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Brand    string `json:"brand"`
	PriceTHB int64  `json:"price_thb"`
	IsActive bool   `json:"is_active"`
}

type Filter struct {
	Category string
	Search   string
	Page     int
	PageSize int
}

type Page struct {
	Items    []Product `json:"items"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	Total    int       `json:"total"`
}

// Repository implementations must filter inactive products before pagination
// and counting. Results must have a stable order (ascending UUID).
type Repository interface {
	ListActive(context.Context, Filter) ([]Product, int, error)
	FindActiveByID(context.Context, string) (Product, error)
}

// ValidUUID accepts the canonical UUID shape without imposing a UUID version.
func ValidUUID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, c := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}
