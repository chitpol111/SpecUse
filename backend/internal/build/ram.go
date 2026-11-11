package build

import (
	"math"
	"sort"
	"strings"

	"specuse/backend/internal/catalog"
)

func multiply(a, b int64) (int64, error) {
	if a < 0 || b < 0 {
		return 0, ErrInvalidInput
	}
	if b != 0 && a > math.MaxInt64/b {
		return 0, ErrOverflow
	}
	return a * b, nil
}

func add(a, b int64) (int64, error) {
	if a < 0 || b < 0 {
		return 0, ErrInvalidInput
	}
	if a > math.MaxInt64-b {
		return 0, ErrOverflow
	}
	return a + b, nil
}

// SelectRAM assumes no inventory limit, since stock is not in the contract.
// For sizes 1 and 2, only the cheapest SKU of each size can be optimal.
// Equal-price SKUs use the smallest UUID. Equal-cost combinations use fewer
// packages, then the same UUID rule. Runtime does not grow with module count.
func SelectRAM(item RAMItem, candidates []Component) (RAMSelection, error) {
	if !catalog.ValidUUID(item.RAMModelID) || item.RequiredModules < 1 ||
		item.OwnedModules < 0 || item.OwnedModules > item.RequiredModules {
		return RAMSelection{}, ErrInvalidInput
	}
	item.RAMModelID = strings.ToLower(item.RAMModelID)
	selection := RAMSelection{RAMItem: item, PurchaseModules: item.RequiredModules - item.OwnedModules, Packages: []RAMPurchase{}}
	if selection.PurchaseModules == 0 {
		return selection, nil
	}
	best := make(map[int64]Component)
	for _, candidate := range candidates {
		if !candidate.Product.IsActive || candidate.Product.Category != "ram" ||
			strings.ToLower(candidate.RAMModelID) != item.RAMModelID {
			continue
		}
		size := candidate.Specs.ModulesPerPackage
		if (size != 1 && size != 2) || candidate.Product.PriceTHB < 0 || !catalog.ValidUUID(candidate.Product.ID) {
			return RAMSelection{}, ErrInvalidInput
		}
		candidate.Product.ID = strings.ToLower(candidate.Product.ID)
		previous, exists := best[size]
		if !exists || candidate.Product.PriceTHB < previous.Product.PriceTHB ||
			(candidate.Product.PriceTHB == previous.Product.PriceTHB && candidate.Product.ID < previous.Product.ID) {
			best[size] = candidate
		}
	}
	one, hasOne := best[1]
	two, hasTwo := best[2]
	if !hasOne && !hasTwo {
		return RAMSelection{}, ErrUnavailable
	}
	need := selection.PurchaseModules
	counts := map[int64]int64{}
	switch {
	case !hasOne:
		counts[2] = need/2 + need%2
	case !hasTwo:
		counts[1] = need
	default:
		// Exact counts always take priority, even if excess would cost less.
		// Short-circuit before doubling a price to prevent overflow.
		preferPair := one.Product.PriceTHB > math.MaxInt64/2 || two.Product.PriceTHB <= 2*one.Product.PriceTHB
		if preferPair {
			counts[2], counts[1] = need/2, need%2
		} else {
			counts[1] = need
		}
	}
	for _, size := range []int64{1, 2} {
		count := counts[size]
		if count == 0 {
			continue
		}
		product := best[size].Product
		price, err := multiply(count, product.PriceTHB)
		if err != nil {
			return RAMSelection{}, err
		}
		modules, err := multiply(count, size)
		if err != nil {
			return RAMSelection{}, err
		}
		selection.TotalPriceTHB, err = add(selection.TotalPriceTHB, price)
		if err != nil {
			return RAMSelection{}, err
		}
		selection.SuppliedModules, err = add(selection.SuppliedModules, modules)
		if err != nil {
			return RAMSelection{}, err
		}
		selection.Packages = append(selection.Packages, RAMPurchase{ProductID: product.ID, Quantity: count, ModulesPerPackage: size, UnitPriceTHB: product.PriceTHB})
	}
	selection.ExcessModules = selection.SuppliedModules - need
	sort.Slice(selection.Packages, func(i, j int) bool { return selection.Packages[i].ProductID < selection.Packages[j].ProductID })
	return selection, nil
}
