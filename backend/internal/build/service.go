package build

import (
	"context"
	"strings"

	"specuse/backend/internal/catalog"
)

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

// Validate accepts references/counts only; prices, specs and active status
// always come from the trusted repository, shared with catalog reads.
func (s *Service) Validate(ctx context.Context, items []Item, ramItems []RAMItem) (Result, error) {
	return s.validate(ctx, items, ramItems, false)
}

// ValidateExtended opts into the agreed coverage/status contract. Legacy
// validation deliberately keeps its previous scope, limits and JSON behavior.
func (s *Service) ValidateExtended(ctx context.Context, items []Item, ramItems []RAMItem) (Result, error) {
	return s.validate(ctx, items, ramItems, true)
}

func (s *Service) validate(ctx context.Context, items []Item, ramItems []RAMItem, extended bool) (Result, error) {
	seen := map[string]bool{}
	for _, item := range items {
		id := strings.ToLower(item.ProductID)
		if !catalog.ValidUUID(id) || item.Quantity < 1 || item.OwnedQuantity < 0 || item.OwnedQuantity > item.Quantity || seen[id] {
			return Result{}, ErrInvalidInput
		}
		seen[id] = true
	}
	seen = map[string]bool{}
	for _, item := range ramItems {
		id := strings.ToLower(item.RAMModelID)
		if !catalog.ValidUUID(id) || item.RequiredModules < 1 || item.OwnedModules < 0 || item.OwnedModules > item.RequiredModules || seen[id] {
			return Result{}, ErrInvalidInput
		}
		seen[id] = true
	}
	snapshot, err := s.repository.LoadSnapshot(ctx)
	if err != nil {
		return Result{}, err
	}
	products := map[string]Component{}
	for _, component := range snapshot.Products {
		products[component.Product.ID] = component
	}
	models := map[string]RAMModel{}
	for _, model := range snapshot.RAMModels {
		models[model.ID] = model
	}
	if extended {
		// Resolve every reference before limits/pricing. A missing or inactive
		// product/model must remain 404, including fully owned devices.
		for _, item := range items {
			component, exists := products[strings.ToLower(item.ProductID)]
			if !exists || !component.Product.IsActive {
				return Result{}, ErrUnavailable
			}
			if component.Product.Category == "ram" {
				return Result{}, ErrInvalidInput
			}
		}
		for _, item := range ramItems {
			id := strings.ToLower(item.RAMModelID)
			if _, exists := models[id]; !exists {
				return Result{}, ErrUnavailable
			}
			if item.RequiredModules > item.OwnedModules {
				available := false
				for _, p := range snapshot.Products {
					if p.Product.Category == "ram" && p.Product.IsActive && p.RAMModelID == id {
						available = true
						break
					}
				}
				if !available {
					return Result{}, ErrUnavailable
				}
			}
		}
		counts := map[string]int64{}
		for _, item := range items {
			category := products[strings.ToLower(item.ProductID)].Product.Category
			limit := int64(0)
			if category == "gpu" {
				limit = 1
			}
			if category == "storage" {
				limit = 8
			}
			if limit > 0 {
				// Bounded subtraction prevents count overflow, even at MaxInt64.
				if item.Quantity > limit-counts[category] {
					return Result{}, ErrUnsupportedScope
				}
				counts[category] += item.Quantity
			}
		}
	}
	result := Result{Items: []PricedItem{}, RAMSelections: []RAMSelection{}}
	selected := []Component{}
	selectedModels := []RAMModel{}
	for _, item := range items {
		item.ProductID = strings.ToLower(item.ProductID)
		component, exists := products[item.ProductID]
		if !exists || !component.Product.IsActive {
			return Result{}, ErrUnavailable
		}
		if component.Product.Category == "ram" {
			return Result{}, ErrInvalidInput // RAM must use model/module requirements.
		}
		purchaseQuantity := item.Quantity - item.OwnedQuantity
		price, err := multiply(purchaseQuantity, component.Product.PriceTHB)
		if err != nil {
			return Result{}, err
		}
		result.TotalPriceTHB, err = add(result.TotalPriceTHB, price)
		if err != nil {
			return Result{}, err
		}
		result.Items = append(result.Items, PricedItem{Item: item, PurchaseQuantity: purchaseQuantity, UnitPriceTHB: component.Product.PriceTHB, PurchasePriceTHB: price})
		selected = append(selected, component)
	}
	for _, item := range ramItems {
		item.RAMModelID = strings.ToLower(item.RAMModelID)
		model, exists := models[item.RAMModelID]
		if !exists {
			return Result{}, ErrUnavailable
		}
		selection, err := SelectRAM(item, snapshot.Products)
		if err != nil {
			return Result{}, err
		}
		result.TotalPriceTHB, err = add(result.TotalPriceTHB, selection.TotalPriceTHB)
		if err != nil {
			return Result{}, err
		}
		result.RAMSelections = append(result.RAMSelections, selection)
		selectedModels = append(selectedModels, model)
	}
	result.Issues = CheckCompatibility(selected, selectedModels)
	result.Compatible = len(result.Issues) == 0
	if extended {
		ramSKUs := []Component{}
		for _, selection := range result.RAMSelections {
			for _, pack := range selection.Packages {
				ramSKUs = append(ramSKUs, products[pack.ProductID])
			}
		}
		result.Compatibility = phaseACompatibility(selected, selectedModels, ramSKUs)
		result.Compatible = result.Compatibility.Status == "passed"
		proven := []Issue{}
		for _, issue := range result.Issues {
			if issue.Code == "CPU_SOCKET_MISMATCH" || issue.Code == "RAM_TYPE_MISMATCH" {
				proven = append(proven, issue)
			}
		}
		result.Issues = proven
	}
	return result, nil
}
