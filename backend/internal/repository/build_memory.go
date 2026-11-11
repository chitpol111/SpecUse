package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/hardware"
)

// BuildMemory serves public catalog reads and build validation from the same
// immutable data, including compatibility metadata for catalog responses.
type BuildMemory struct {
	snapshot build.Snapshot
	catalog  *Memory
	byID     map[string]build.Component
}

var _ build.Repository = (*BuildMemory)(nil)
var _ catalog.Repository = (*BuildMemory)(nil)

func NewBuildMemory(source build.Snapshot) (*BuildMemory, error) {
	return newBuildMemory(source, false)
}

// NewLocalBuildMemory permits explicitly marked synthetic provenance. The
// application calls this only after enforcing APP_ENV=local and a fixture path.
func NewLocalBuildMemory(source build.Snapshot) (*BuildMemory, error) {
	return newBuildMemory(source, true)
}

func newBuildMemory(source build.Snapshot, allowSynthetic bool) (*BuildMemory, error) {
	owned, err := hardware.Clone(source)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid metadata: %v", build.ErrInvalidInput, err)
	}
	snapshot := build.Snapshot{
		Products: append([]build.Component{}, owned.Products...), RAMModels: append([]build.RAMModel{}, owned.RAMModels...),
	}
	models := map[string]string{}
	for i := range snapshot.RAMModels {
		model := &snapshot.RAMModels[i]
		model.ID = strings.ToLower(model.ID)
		model.RAMType = strings.TrimSpace(model.RAMType)
		if !catalog.ValidUUID(model.ID) || model.RAMType == "" || models[model.ID] != "" {
			return nil, fmt.Errorf("%w: invalid or duplicate RAM model", build.ErrInvalidInput)
		}
		models[model.ID] = model.RAMType
	}
	base := make([]catalog.Product, 0, len(snapshot.Products))
	for i := range snapshot.Products {
		component := &snapshot.Products[i]
		component.Product.ID = strings.ToLower(component.Product.ID)
		component.RAMModelID = strings.ToLower(component.RAMModelID)
		component.Specs.CPUSocket = strings.TrimSpace(component.Specs.CPUSocket)
		component.Specs.RAMType = strings.TrimSpace(component.Specs.RAMType)
		product, specs := component.Product, component.Specs
		if product.Category != strings.ToLower(strings.TrimSpace(product.Category)) {
			return nil, fmt.Errorf("%w: category must be lowercase", build.ErrInvalidInput)
		}
		valid := false
		switch product.Category {
		case "cpu":
			valid = specs.CPUSocket != "" && specs.RAMType == "" && specs.ModulesPerPackage == 0 && component.RAMModelID == "" && specs.GPU == nil && specs.Storage == nil && specs.Motherboard == nil
		case "motherboard":
			valid = specs.CPUSocket != "" && specs.RAMType != "" && specs.ModulesPerPackage == 0 && component.RAMModelID == "" && specs.GPU == nil && specs.Storage == nil
		case "ram":
			valid = specs.CPUSocket == "" && specs.RAMType != "" && (specs.ModulesPerPackage == 1 || specs.ModulesPerPackage == 2) && models[component.RAMModelID] == specs.RAMType && specs.GPU == nil && specs.Storage == nil && specs.Motherboard == nil
		case "gpu":
			valid = specs.CPUSocket == "" && specs.RAMType == "" && specs.ModulesPerPackage == 0 && component.RAMModelID == "" && specs.Storage == nil && specs.Motherboard == nil
		case "storage":
			valid = specs.CPUSocket == "" && specs.RAMType == "" && specs.ModulesPerPackage == 0 && component.RAMModelID == "" && specs.GPU == nil && specs.Motherboard == nil
		default:
			valid = specs == (build.Specs{}) && component.RAMModelID == ""
		}
		if !valid {
			return nil, fmt.Errorf("%w: invalid compatibility metadata", build.ErrInvalidInput)
		}
		for _, err := range []error{hardware.ValidateGPU(specs.GPU, allowSynthetic), hardware.ValidateStorage(specs.Storage, allowSynthetic), hardware.ValidateMotherboard(specs.Motherboard, allowSynthetic)} {
			if err != nil {
				return nil, fmt.Errorf("%w: %v", build.ErrInvalidInput, err)
			}
		}
		base = append(base, product)
	}
	baseRepository, err := NewMemory(base)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", build.ErrInvalidInput, err)
	}
	sort.Slice(snapshot.Products, func(i, j int) bool { return snapshot.Products[i].Product.ID < snapshot.Products[j].Product.ID })
	sort.Slice(snapshot.RAMModels, func(i, j int) bool { return snapshot.RAMModels[i].ID < snapshot.RAMModels[j].ID })
	byID := make(map[string]build.Component, len(snapshot.Products))
	for _, component := range snapshot.Products {
		byID[component.Product.ID] = component
	}
	return &BuildMemory{snapshot: snapshot, catalog: baseRepository, byID: byID}, nil
}

func (m *BuildMemory) ListActive(ctx context.Context, filter catalog.Filter) ([]catalog.ProductDetails, int, error) {
	items, total, err := m.catalog.ListActive(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i], err = m.withDetails(items[i])
		if err != nil {
			return nil, 0, err
		}
	}
	return items, total, nil
}

func (m *BuildMemory) FindActiveByID(ctx context.Context, id string) (catalog.ProductDetails, error) {
	product, err := m.catalog.FindActiveByID(ctx, id)
	if err != nil {
		return catalog.ProductDetails{}, err
	}
	return m.withDetails(product)
}

func (m *BuildMemory) withDetails(product catalog.ProductDetails) (catalog.ProductDetails, error) {
	component, err := hardware.Clone(m.byID[product.ID])
	if err != nil {
		return catalog.ProductDetails{}, err
	}
	switch product.Category {
	case "cpu", "motherboard", "ram":
		// Copy into a new DTO for each read. Callers cannot mutate the specs
		// used by validation or another public response through this pointer.
		product.Specs = &catalog.Specs{CPUSocket: component.Specs.CPUSocket,
			RAMType: component.Specs.RAMType, ModulesPerPackage: component.Specs.ModulesPerPackage, Motherboard: component.Specs.Motherboard}
		if product.Category == "motherboard" && product.Specs.Motherboard == nil {
			product.Specs.Motherboard = &hardware.Motherboard{}
		}
		if product.Category == "ram" {
			product.RAMModelID = component.RAMModelID
		}
	case "gpu":
		metadata := component.Specs.GPU
		if metadata == nil {
			metadata = &hardware.GPU{}
		}
		product.Specs = &catalog.Specs{GPU: metadata}
	case "storage":
		metadata := component.Specs.Storage
		if metadata == nil {
			metadata = &hardware.Storage{}
		}
		product.Specs = &catalog.Specs{Storage: metadata}
	}
	return product, nil
}

func (m *BuildMemory) LoadSnapshot(ctx context.Context) (build.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return build.Snapshot{}, err
	}
	return hardware.Clone(m.snapshot)
}
