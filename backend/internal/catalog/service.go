package catalog

import (
	"context"
	"fmt"
	"strings"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// ValidateFilter also guards the offset arithmetic used by repositories.
func ValidateFilter(filter Filter) error {
	maxInt := int(^uint(0) >> 1)
	if filter.Page < 1 || filter.PageSize < 1 || filter.Page-1 > maxInt/filter.PageSize {
		return fmt.Errorf("%w: invalid pagination", ErrInvalidInput)
	}
	return nil
}

func (s *Service) List(ctx context.Context, filter Filter) (Page, error) {
	if err := ValidateFilter(filter); err != nil {
		return Page{}, err
	}
	filter.Category = strings.TrimSpace(filter.Category)
	filter.Search = strings.TrimSpace(filter.Search)
	items, total, err := s.repository.ListActive(ctx, filter)
	if err != nil {
		return Page{}, err
	}
	if items == nil {
		items = []Product{}
	}
	return Page{Items: items, Page: filter.Page, PageSize: filter.PageSize, Total: total}, nil
}

func (s *Service) Get(ctx context.Context, id string) (Product, error) {
	if !ValidUUID(id) {
		return Product{}, fmt.Errorf("%w: invalid UUID", ErrInvalidInput)
	}
	product, err := s.repository.FindActiveByID(ctx, strings.ToLower(id))
	if err != nil {
		return Product{}, err
	}
	if !product.IsActive {
		return Product{}, ErrNotFound
	}
	return product, nil
}
