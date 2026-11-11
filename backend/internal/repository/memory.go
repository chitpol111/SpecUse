package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"specuse/backend/internal/catalog"
)

// Memory is an immutable local adapter. It does not connect to Supabase and
// intentionally has no mutation API. A future database adapter can use the
// same catalog.Repository interface.
type Memory struct {
	products []catalog.Product
}

var _ catalog.Repository = (*Memory)(nil)

func NewMemory(products []catalog.Product) (*Memory, error) {
	result := &Memory{products: make([]catalog.Product, 0, len(products))}
	seen := make(map[string]bool, len(products))
	for _, product := range products {
		if !catalog.ValidUUID(product.ID) || product.PriceTHB < 0 ||
			strings.TrimSpace(product.Name) == "" || strings.TrimSpace(product.Category) == "" ||
			strings.TrimSpace(product.Brand) == "" {
			return nil, fmt.Errorf("%w: invalid catalog product", catalog.ErrInvalidInput)
		}
		product.ID = strings.ToLower(product.ID)
		if seen[product.ID] {
			return nil, fmt.Errorf("%w: duplicate product UUID", catalog.ErrInvalidInput)
		}
		seen[product.ID] = true
		result.products = append(result.products, product)
	}
	sort.Slice(result.products, func(i, j int) bool {
		return result.products[i].ID < result.products[j].ID
	})
	return result, nil
}

func (m *Memory) ListActive(ctx context.Context, filter catalog.Filter) ([]catalog.ProductDetails, int, error) {
	if err := catalog.ValidateFilter(filter); err != nil {
		return nil, 0, err
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	matches := make([]catalog.ProductDetails, 0)
	search := strings.ToLower(filter.Search)
	for _, product := range m.products {
		if !product.IsActive || (filter.Category != "" && product.Category != filter.Category) {
			continue
		}
		// Phase-one search matches product names, case-insensitively.
		if search != "" && !strings.Contains(strings.ToLower(product.Name), search) {
			continue
		}
		matches = append(matches, catalog.ProductDetails{Product: product})
	}
	total := len(matches)
	start := (filter.Page - 1) * filter.PageSize
	if start >= total {
		return []catalog.ProductDetails{}, total, nil
	}
	// Bound the size before addition to avoid integer overflow.
	size := filter.PageSize
	if size > total-start {
		size = total - start
	}
	return matches[start : start+size], total, nil
}

func (m *Memory) FindActiveByID(ctx context.Context, id string) (catalog.ProductDetails, error) {
	if err := ctx.Err(); err != nil {
		return catalog.ProductDetails{}, err
	}
	for _, product := range m.products {
		if product.ID == id && product.IsActive {
			return catalog.ProductDetails{Product: product}, nil
		}
	}
	return catalog.ProductDetails{}, catalog.ErrNotFound
}
