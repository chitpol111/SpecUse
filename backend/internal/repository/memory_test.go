package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"specuse/backend/internal/catalog"
)

func products() []catalog.Product {
	return []catalog.Product{
		{ID: "00000000-0000-0000-0000-000000000003", Name: "Ryzen B", Category: "cpu", Brand: "AMD", PriceTHB: 25000, IsActive: true},
		{ID: "00000000-0000-0000-0000-000000000001", Name: "Ryzen hidden", Category: "cpu", Brand: "AMD", PriceTHB: 1000, IsActive: false},
		{ID: "00000000-0000-0000-0000-000000000004", Name: "Board", Category: "motherboard", Brand: "Example", PriceTHB: 3000, IsActive: true},
		{ID: "00000000-0000-0000-0000-000000000002", Name: "Ryzen A", Category: "cpu", Brand: "AMD", PriceTHB: 2000, IsActive: true},
	}
}

func TestMemoryFiltersBeforeCountingAndPagination(t *testing.T) {
	repo, err := NewMemory(products())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		filter    catalog.Filter
		wantIDs   []string
		wantTotal int
	}{
		{"all", catalog.Filter{Page: 1, PageSize: 20}, []string{"2", "3", "4"}, 3},
		{"category and search", catalog.Filter{Category: "cpu", Search: "rYzEn", Page: 1, PageSize: 1}, []string{"2"}, 2},
		{"second page", catalog.Filter{Category: "cpu", Page: 2, PageSize: 1}, []string{"3"}, 2},
		{"out of range", catalog.Filter{Page: 4, PageSize: 1}, []string{}, 3},
		{"inactive search", catalog.Filter{Search: "hidden", Page: 1, PageSize: 20}, []string{}, 0},
		{"unknown category", catalog.Filter{Category: "unknown", Page: 1, PageSize: 20}, []string{}, 0},
		{"no name match", catalog.Filter{Search: "AMD", Page: 1, PageSize: 20}, []string{}, 0},
		{"large size", catalog.Filter{Page: 1, PageSize: int(^uint(0) >> 1)}, []string{"2", "3", "4"}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, total, err := repo.ListActive(context.Background(), tt.filter)
			if err != nil || total != tt.wantTotal || items == nil {
				t.Fatalf("items=%v total=%d err=%v", items, total, err)
			}
			ids := make([]string, 0, len(items))
			for _, item := range items {
				if !item.IsActive {
					t.Fatal("inactive product leaked")
				}
				ids = append(ids, item.ID[len(item.ID)-1:])
			}
			if !reflect.DeepEqual(ids, tt.wantIDs) {
				t.Fatalf("IDs=%v want=%v", ids, tt.wantIDs)
			}
		})
	}
}

func TestMemoryDetailAndDefensiveCopies(t *testing.T) {
	source := products()
	repo, err := NewMemory(source)
	if err != nil {
		t.Fatal(err)
	}
	source[0].Name = "changed"
	product, err := repo.FindActiveByID(context.Background(), "00000000-0000-0000-0000-000000000003")
	if err != nil || product.Name != "Ryzen B" || product.PriceTHB != 25000 {
		t.Fatalf("product=%+v err=%v", product, err)
	}
	items, _, _ := repo.ListActive(context.Background(), catalog.Filter{Page: 1, PageSize: 20})
	items[0].IsActive = false
	if _, err := repo.FindActiveByID(context.Background(), items[0].ID); err != nil {
		t.Fatalf("caller modified stored data: %v", err)
	}
	for _, id := range []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000099"} {
		if _, err := repo.FindActiveByID(context.Background(), id); !errors.Is(err, catalog.ErrNotFound) {
			t.Fatalf("id=%s err=%v", id, err)
		}
	}
}

func TestMemoryRejectsInvalidSource(t *testing.T) {
	valid := products()[0]
	for _, change := range []func(*catalog.Product){
		func(p *catalog.Product) { p.ID = "invalid" },
		func(p *catalog.Product) { p.PriceTHB = -1 },
		func(p *catalog.Product) { p.Name = " " },
		func(p *catalog.Product) { p.Category = "" },
		func(p *catalog.Product) { p.Brand = "" },
	} {
		p := valid
		change(&p)
		if _, err := NewMemory([]catalog.Product{p}); !errors.Is(err, catalog.ErrInvalidInput) {
			t.Errorf("product=%+v err=%v", p, err)
		}
	}
	if _, err := NewMemory([]catalog.Product{valid, valid}); !errors.Is(err, catalog.ErrInvalidInput) {
		t.Fatalf("duplicate UUID: %v", err)
	}
}

func TestMemoryCanceledContext(t *testing.T) {
	repo, _ := NewMemory(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, listErr := repo.ListActive(ctx, catalog.Filter{Page: 1, PageSize: 20})
	_, getErr := repo.FindActiveByID(ctx, "00000000-0000-0000-0000-000000000001")
	if !errors.Is(listErr, context.Canceled) || !errors.Is(getErr, context.Canceled) {
		t.Fatalf("list=%v get=%v", listErr, getErr)
	}
}
