package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"specuse/backend/internal/hardware"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("product not found")
)

// Product retains the original six shared fields used by the build domain.
type Product struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Brand    string `json:"brand"`
	PriceTHB int64  `json:"price_thb"`
	IsActive bool   `json:"is_active"`
}

// Specs contains only the compatibility fields agreed with the frontend.
// Unrelated fields are omitted instead of supplying guessed or zero values.
type Specs struct {
	CPUSocket         string                `json:"cpu_socket,omitempty"`
	RAMType           string                `json:"ram_type,omitempty"`
	ModulesPerPackage int64                 `json:"modules_per_package,omitempty"`
	GPU               *hardware.GPU         `json:"-"`
	Storage           *hardware.Storage     `json:"-"`
	Motherboard       *hardware.Motherboard `json:"-"`
}

// MarshalJSON flattens the category's typed metadata under specs, never under
// invented GPU/storage wrapper keys. CPU/RAM retain their legacy JSON fields.
func (s Specs) MarshalJSON() ([]byte, error) {
	if s.GPU != nil {
		return json.Marshal(s.GPU)
	}
	if s.Storage != nil {
		return json.Marshal(s.Storage)
	}
	if s.Motherboard != nil {
		return json.Marshal(struct {
			CPUSocket string `json:"cpu_socket"`
			RAMType   string `json:"ram_type"`
			*hardware.Motherboard
		}{s.CPUSocket, s.RAMType, s.Motherboard})
	}
	type legacy Specs
	return json.Marshal(legacy(s))
}

// ProductDetails is an additive public view. IDs, prices and the six original
// fields keep their existing meaning; RAMModelID identifies a model, not a SKU.
type ProductDetails struct {
	Product
	Specs      *Specs `json:"specs,omitempty"`
	RAMModelID string `json:"ram_model_id,omitempty"`
}

type Filter struct {
	Category string
	Search   string
	Page     int
	PageSize int
}

type Page struct {
	Items    []ProductDetails `json:"items"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	Total    int              `json:"total"`
}

// Repository implementations must filter inactive products before pagination
// and counting. Results must have a stable order (ascending UUID).
type Repository interface {
	ListActive(context.Context, Filter) ([]ProductDetails, int, error)
	FindActiveByID(context.Context, string) (ProductDetails, error)
}

// ValidUUID accepts the canonical UUID shape without imposing a UUID version.
func ValidUUID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, c := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}
