package build

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"specuse/backend/internal/catalog"
)

const cpuID = "30000000-0000-0000-0000-000000000001"
const boardID = "30000000-0000-0000-0000-000000000002"

func testSnapshot() Snapshot {
	return Snapshot{
		Products: []Component{
			{Product: catalog.Product{ID: cpuID, Name: "CPU", Category: "cpu", Brand: "Example", PriceTHB: 5000, IsActive: true}, Specs: Specs{CPUSocket: "AM5"}},
			{Product: catalog.Product{ID: boardID, Name: "Board", Category: "motherboard", Brand: "Example", PriceTHB: 3000, IsActive: true}, Specs: Specs{CPUSocket: "AM5", RAMType: "DDR5"}},
			ram(singleID, 1, 1000), ram(pairID, 2, 1500),
		},
		RAMModels: []RAMModel{{modelID, "DDR5"}},
	}
}

type stubRepository struct {
	snapshot Snapshot
	err      error
	calls    int
}

func (r *stubRepository) LoadSnapshot(ctx context.Context) (Snapshot, error) {
	r.calls++
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	return r.snapshot, r.err
}

func TestValidateUsesOneTrustedSnapshotAndPurchaseQuantities(t *testing.T) {
	repo := &stubRepository{snapshot: testSnapshot()}
	result, err := NewService(repo).Validate(context.Background(), []Item{{cpuID, 1, 1}, {boardID, 2, 1}}, []RAMItem{{modelID, 4, 2}})
	if err != nil || !result.Compatible || len(result.Issues) != 0 || result.TotalPriceTHB != 4500 || repo.calls != 1 {
		t.Fatalf("result=%+v calls=%d err=%v", result, repo.calls, err)
	}
	if result.Items[0].PurchasePriceTHB != 0 || result.Items[1].PurchasePriceTHB != 3000 || result.RAMSelections[0].TotalPriceTHB != 1500 {
		t.Fatalf("owned quantities must not be charged: %+v", result)
	}
	// A new authoritative catalog price changes the total without client input.
	repo.snapshot.Products[1].Product.PriceTHB = 4000
	updated, err := NewService(repo).Validate(context.Background(), []Item{{boardID, 1, 0}}, nil)
	if err != nil || updated.TotalPriceTHB != 4000 {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
}

func TestValidateIncompleteDraftAndOwnedRAM(t *testing.T) {
	snapshot := testSnapshot()
	for i := range snapshot.Products {
		snapshot.Products[i].Product.IsActive = false
	}
	// No active RAM packages are needed if all requested modules are owned.
	result, err := NewService(&stubRepository{snapshot: snapshot}).Validate(context.Background(), nil, []RAMItem{{modelID, 2, 2}})
	if err != nil || !result.Compatible || result.TotalPriceTHB != 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	result, err = NewService(&stubRepository{snapshot: snapshot}).Validate(context.Background(), nil, nil)
	if err != nil || !result.Compatible || result.Items == nil || result.RAMSelections == nil || result.Issues == nil {
		t.Fatalf("partial/empty draft evaluation must not invent completeness: %+v %v", result, err)
	}
}

func TestValidateChecksOwnedRAMType(t *testing.T) {
	snapshot := testSnapshot()
	snapshot.RAMModels[0].RAMType = "DDR4"
	result, err := NewService(&stubRepository{snapshot: snapshot}).Validate(context.Background(), []Item{{boardID, 1, 0}}, []RAMItem{{modelID, 2, 2}})
	if err != nil || result.Compatible || len(result.Issues) != 1 || result.Issues[0].Code != "RAM_TYPE_MISMATCH" {
		t.Fatalf("owned RAM still affects compatibility: %+v %v", result, err)
	}
}

func TestValidateRejectsInvalidRequirementsBeforeRepository(t *testing.T) {
	for _, items := range [][]Item{
		{{"bad", 1, 0}}, {{cpuID, 0, 0}}, {{cpuID, 1, -1}}, {{cpuID, 1, 2}}, {{cpuID, 1, 0}, {cpuID, 1, 0}},
	} {
		repo := &stubRepository{}
		_, err := NewService(repo).Validate(context.Background(), items, nil)
		if !errors.Is(err, ErrInvalidInput) || repo.calls != 0 {
			t.Fatalf("items=%+v calls=%d err=%v", items, repo.calls, err)
		}
	}
	for _, items := range [][]RAMItem{
		{{"bad", 1, 0}}, {{modelID, 0, 0}}, {{modelID, 1, -1}}, {{modelID, 1, 2}}, {{modelID, 1, 0}, {modelID, 1, 0}},
	} {
		repo := &stubRepository{}
		_, err := NewService(repo).Validate(context.Background(), nil, items)
		if !errors.Is(err, ErrInvalidInput) || repo.calls != 0 {
			t.Fatalf("items=%+v calls=%d err=%v", items, repo.calls, err)
		}
	}
}

func TestValidateUnavailableReferencesAndRAMRouting(t *testing.T) {
	for _, tt := range []struct {
		name     string
		items    []Item
		rams     []RAMItem
		inactive bool
		want     error
	}{
		{"unknown product", []Item{{"30000000-0000-0000-0000-000000000009", 1, 0}}, nil, false, ErrUnavailable},
		{"inactive product", []Item{{cpuID, 1, 0}}, nil, true, ErrUnavailable},
		{"RAM in regular items", []Item{{singleID, 1, 0}}, nil, false, ErrInvalidInput},
		{"unknown RAM model", nil, []RAMItem{{"10000000-0000-0000-0000-000000000009", 1, 0}}, false, ErrUnavailable},
		{"no active RAM", nil, []RAMItem{{modelID, 1, 0}}, true, ErrUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := testSnapshot()
			if tt.inactive {
				for i := range snapshot.Products {
					snapshot.Products[i].Product.IsActive = false
				}
			}
			_, err := NewService(&stubRepository{snapshot: snapshot}).Validate(context.Background(), tt.items, tt.rams)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v want %v", err, tt.want)
			}
		})
	}
}

func TestValidateRepositoryErrorsAndOverflow(t *testing.T) {
	want := errors.New("repository failed")
	_, err := NewService(&stubRepository{err: want}).Validate(context.Background(), nil, nil)
	if !errors.Is(err, want) {
		t.Fatalf("got %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = NewService(&stubRepository{}).Validate(ctx, nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	snapshot := testSnapshot()
	snapshot.Products[0].Product.PriceTHB = math.MaxInt64
	s := NewService(&stubRepository{snapshot: snapshot})
	for _, items := range [][]Item{{{cpuID, 2, 0}}, {{cpuID, 1, 0}, {boardID, 1, 0}}} {
		if _, err := s.Validate(context.Background(), items, nil); !errors.Is(err, ErrOverflow) {
			t.Fatalf("items=%+v err=%v", items, err)
		}
	}
	if _, err := s.Validate(context.Background(), []Item{{cpuID, 1, 0}}, []RAMItem{{modelID, 1, 0}}); !errors.Is(err, ErrOverflow) {
		t.Fatalf("RAM sum overflow: %v", err)
	}
}

func TestValidateNormalizesUUIDCase(t *testing.T) {
	snapshot := testSnapshot()
	id := "abcdefab-0000-0000-0000-000000000001"
	snapshot.Products[0].Product.ID = id
	result, err := NewService(&stubRepository{snapshot: snapshot}).Validate(context.Background(), []Item{{strings.ToUpper(id), 1, 0}}, nil)
	if err != nil || result.Items[0].ProductID != id {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
