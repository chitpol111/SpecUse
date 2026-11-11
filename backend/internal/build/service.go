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
	return result, nil
}
