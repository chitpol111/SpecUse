package catalog

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

type stubRepository struct {
	filter Filter
	id     string
	called bool
	items  []ProductDetails
	total  int
	item   ProductDetails
	err    error
}

func (r *stubRepository) ListActive(_ context.Context, filter Filter) ([]ProductDetails, int, error) {
	r.called, r.filter = true, filter
	return r.items, r.total, r.err
}

func (r *stubRepository) FindActiveByID(_ context.Context, id string) (ProductDetails, error) {
	r.called, r.id = true, id
	return r.item, r.err
}

func TestListValidatesBeforeRepository(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, filter := range []Filter{
		{Page: 0, PageSize: 20}, {Page: -1, PageSize: 20},
		{Page: 1, PageSize: 0}, {Page: 1, PageSize: -1},
		{Page: maxInt, PageSize: maxInt},
	} {
		t.Run(fmt.Sprint(filter), func(t *testing.T) {
			repo := &stubRepository{}
			_, err := NewService(repo).List(context.Background(), filter)
			if !errors.Is(err, ErrInvalidInput) || repo.called {
				t.Fatalf("err=%v repository called=%v", err, repo.called)
			}
		})
	}
}

func TestListNormalizesAndReturnsNonNilItems(t *testing.T) {
	repo := &stubRepository{total: 3}
	page, err := NewService(repo).List(context.Background(), Filter{
		Category: " cpu ", Search: " Ryzen ", Page: 2, PageSize: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.filter.Category != "cpu" || repo.filter.Search != "Ryzen" {
		t.Fatalf("unexpected filter: %+v", repo.filter)
	}
	if page.Items == nil || page.Page != 2 || page.PageSize != 20 || page.Total != 3 {
		t.Fatalf("unexpected page: %+v", page)
	}
}

func TestGetValidatesAndNormalizesUUID(t *testing.T) {
	for _, id := range []string{"", "not-a-uuid", "00000000-0000-0000-0000-00000000000g", "00000000000000000000000000000000", "00000000_0000-0000-0000-000000000001"} {
		repo := &stubRepository{}
		_, err := NewService(repo).Get(context.Background(), id)
		if !errors.Is(err, ErrInvalidInput) || repo.called {
			t.Errorf("id=%q err=%v repository called=%v", id, err, repo.called)
		}
	}
	repo := &stubRepository{item: ProductDetails{Product: Product{IsActive: true}}}
	_, err := NewService(repo).Get(context.Background(), "ABCDEFAB-0000-0000-0000-000000000001")
	if err != nil || repo.id != "abcdefab-0000-0000-0000-000000000001" {
		t.Fatalf("id=%q err=%v", repo.id, err)
	}
}

func TestGetRejectsInactiveRepositoryResult(t *testing.T) {
	_, err := NewService(&stubRepository{item: ProductDetails{Product: Product{IsActive: false}}}).
		Get(context.Background(), "00000000-0000-0000-0000-000000000001")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestServicePropagatesRepositoryErrors(t *testing.T) {
	want := errors.New("database unavailable")
	s := NewService(&stubRepository{err: want})
	_, listErr := s.List(context.Background(), Filter{Page: 1, PageSize: 20})
	_, getErr := s.Get(context.Background(), "00000000-0000-0000-0000-000000000001")
	if !errors.Is(listErr, want) || !errors.Is(getErr, want) {
		t.Fatalf("list=%v get=%v", listErr, getErr)
	}
}
