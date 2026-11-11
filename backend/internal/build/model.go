// Package build contains internal validation models. These types are not
// public HTTP DTOs; the phase-one catalog JSON remains unchanged.
package build

import (
	"context"
	"errors"

	"specuse/backend/internal/catalog"
)

var (
	ErrInvalidInput = errors.New("invalid build input")
	ErrUnavailable  = errors.New("product or RAM package unavailable")
	ErrOverflow     = errors.New("build integer overflow")
)

type Specs struct {
	CPUSocket         string
	RAMType           string
	ModulesPerPackage int64
}

type Component struct {
	Product    catalog.Product
	Specs      Specs
	RAMModelID string
}

type RAMModel struct {
	ID      string
	RAMType string
}

// Snapshot provides prices, active status and specs from one trusted read.
// A future database adapter must provide a consistent read/transaction.
type Snapshot struct {
	Products  []Component
	RAMModels []RAMModel
}

type Repository interface {
	LoadSnapshot(context.Context) (Snapshot, error)
}

type Item struct {
	ProductID     string
	Quantity      int64
	OwnedQuantity int64
}

type RAMItem struct {
	RAMModelID      string
	RequiredModules int64
	OwnedModules    int64
}

type PricedItem struct {
	Item
	PurchaseQuantity int64
	UnitPriceTHB     int64
	PurchasePriceTHB int64
}

type RAMPurchase struct {
	ProductID         string
	Quantity          int64
	ModulesPerPackage int64
	UnitPriceTHB      int64
}

type RAMSelection struct {
	RAMItem
	PurchaseModules int64
	SuppliedModules int64
	ExcessModules   int64
	Packages        []RAMPurchase
	TotalPriceTHB   int64
}

type Issue struct {
	Code    string
	LeftID  string
	RightID string
}

type Result struct {
	// Compatible describes only the initial pairwise checks. It is not a
	// completeness check or permission to publish a build.
	Compatible    bool
	Issues        []Issue
	Items         []PricedItem
	RAMSelections []RAMSelection
	TotalPriceTHB int64
}
