package build

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"specuse/backend/internal/catalog"
)

const modelID = "10000000-0000-0000-0000-000000000001"

func ram(id string, size, price int64) Component {
	return Component{Product: catalog.Product{ID: id, Name: "Test RAM", Category: "ram", Brand: "Example", PriceTHB: price, IsActive: true},
		Specs: Specs{RAMType: "DDR5", ModulesPerPackage: size}, RAMModelID: modelID}
}

const singleID = "20000000-0000-0000-0000-000000000001"
const pairID = "20000000-0000-0000-0000-000000000002"

func TestRAMSelectionPriorities(t *testing.T) {
	for _, tt := range []struct {
		name                               string
		required, owned                    int64
		candidates                         []Component
		wantModules, wantExcess, wantPrice int64
		wantPackages                       []RAMPurchase
	}{
		{"exact before cheaper excess", 3, 0, []Component{ram(singleID, 1, 100), ram(pairID, 2, 1)}, 3, 0, 101,
			[]RAMPurchase{{singleID, 1, 1, 100}, {pairID, 1, 2, 1}}},
		{"minimal excess", 3, 0, []Component{ram(pairID, 2, 10)}, 4, 1, 20, []RAMPurchase{{pairID, 2, 2, 10}}},
		{"singles cheaper", 4, 0, []Component{ram(singleID, 1, 4), ram(pairID, 2, 9)}, 4, 0, 16, []RAMPurchase{{singleID, 4, 1, 4}}},
		{"pairs cheaper", 4, 0, []Component{ram(singleID, 1, 5), ram(pairID, 2, 8)}, 4, 0, 16, []RAMPurchase{{pairID, 2, 2, 8}}},
		{"owned deducted", 4, 3, []Component{ram(singleID, 1, 9), ram(pairID, 2, 1)}, 1, 0, 9, []RAMPurchase{{singleID, 1, 1, 9}}},
		{"all owned no active stock needed", 4, 4, nil, 0, 0, 0, []RAMPurchase{}},
		{"equal price fewer packages", 4, 0, []Component{ram(singleID, 1, 5), ram(pairID, 2, 10)}, 4, 0, 20, []RAMPurchase{{pairID, 2, 2, 10}}},
		{"free packages still finite", 3, 0, []Component{ram(singleID, 1, 0), ram(pairID, 2, 0)}, 3, 0, 0,
			[]RAMPurchase{{singleID, 1, 1, 0}, {pairID, 1, 2, 0}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SelectRAM(RAMItem{modelID, tt.required, tt.owned}, tt.candidates)
			if err != nil || result.SuppliedModules != tt.wantModules || result.ExcessModules != tt.wantExcess || result.TotalPriceTHB != tt.wantPrice || !reflect.DeepEqual(result.Packages, tt.wantPackages) {
				t.Fatalf("result=%+v err=%v want packages=%+v", result, err, tt.wantPackages)
			}
		})
	}
}

func TestRAMSkipsInactiveAndOtherModelsAndBreaksSKUTies(t *testing.T) {
	inactive := ram("20000000-0000-0000-0000-000000000000", 1, 0)
	inactive.Product.IsActive = false
	other := ram("20000000-0000-0000-0000-000000000009", 1, 0)
	other.RAMModelID = "10000000-0000-0000-0000-000000000002"
	candidates := []Component{ram(pairID, 1, 5), inactive, other, ram(singleID, 1, 5)}
	for i := 0; i < len(candidates); i++ {
		result, err := SelectRAM(RAMItem{modelID, 2, 0}, candidates)
		if err != nil || len(result.Packages) != 1 || result.Packages[0].ProductID != singleID {
			t.Fatalf("selection must use lowest UUID active SKU of same model: %+v %v", result, err)
		}
		candidates = append(candidates[1:], candidates[0])
	}
}

func TestRAMRejectsInvalidInputAndOverflow(t *testing.T) {
	for _, item := range []RAMItem{{"bad", 2, 0}, {modelID, 0, 0}, {modelID, 1, -1}, {modelID, 1, 2}} {
		if _, err := SelectRAM(item, nil); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("item=%+v err=%v", item, err)
		}
	}
	for _, candidate := range []Component{ram(singleID, 3, 1), ram(singleID, 1, -1), ram("bad", 1, 1)} {
		if _, err := SelectRAM(RAMItem{modelID, 1, 0}, []Component{candidate}); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("candidate=%+v err=%v", candidate, err)
		}
	}
	if _, err := SelectRAM(RAMItem{modelID, 2, 0}, nil); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("no purchasable package: %v", err)
	}
	for _, tt := range []struct {
		required  int64
		candidate Component
	}{
		{2, ram(singleID, 1, math.MaxInt64)},
		{math.MaxInt64, ram(pairID, 2, 0)},
	} {
		if _, err := SelectRAM(RAMItem{modelID, tt.required, 0}, []Component{tt.candidate}); !errors.Is(err, ErrOverflow) {
			t.Fatalf("expected overflow, got %v", err)
		}
	}
	result, err := SelectRAM(RAMItem{modelID, math.MaxInt64, 0}, []Component{ram(singleID, 1, 1), ram(pairID, 2, 1)})
	if err != nil || result.SuppliedModules != math.MaxInt64 || result.TotalPriceTHB != math.MaxInt64/2+1 {
		t.Fatalf("large counts must work without count-sized allocations: %+v %v", result, err)
	}
}

// Independent exhaustive oracle compares all single/pair combinations for
// small inputs, including missing sizes and zero prices.
func TestRAMSelectionMatchesExhaustiveOracle(t *testing.T) {
	for availability := 1; availability <= 3; availability++ {
		for singlePrice := int64(0); singlePrice <= 5; singlePrice++ {
			for pairPrice := int64(0); pairPrice <= 5; pairPrice++ {
				for need := int64(1); need <= 9; need++ {
					candidates := []Component{}
					if availability&1 != 0 {
						candidates = append(candidates, ram(singleID, 1, singlePrice))
					}
					if availability&2 != 0 {
						candidates = append(candidates, ram(pairID, 2, pairPrice))
					}
					wantExcess, wantCost, wantCount := int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MaxInt64)
					for singles := int64(0); singles <= need+1; singles++ {
						for pairs := int64(0); pairs <= need+1; pairs++ {
							if (availability&1 == 0 && singles != 0) || (availability&2 == 0 && pairs != 0) || singles+2*pairs < need {
								continue
							}
							excess, cost, count := singles+2*pairs-need, singles*singlePrice+pairs*pairPrice, singles+pairs
							if excess < wantExcess || (excess == wantExcess && (cost < wantCost || (cost == wantCost && count < wantCount))) {
								wantExcess, wantCost, wantCount = excess, cost, count
							}
						}
					}
					result, err := SelectRAM(RAMItem{modelID, need + 2, 2}, candidates)
					var count int64
					for _, p := range result.Packages {
						count += p.Quantity
					}
					if err != nil || result.ExcessModules != wantExcess || result.TotalPriceTHB != wantCost || count != wantCount {
						t.Fatalf("availability=%d prices=%d/%d need=%d got=%+v count=%d err=%v want=%d/%d/%d", availability, singlePrice, pairPrice, need, result, count, err, wantExcess, wantCost, wantCount)
					}
				}
			}
		}
	}
}
