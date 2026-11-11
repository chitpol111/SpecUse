package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
)

// BuildMemory serves public catalog reads and build validation from the same
// immutable data. Public catalog DTOs still contain only the original fields.
type BuildMemory struct {
	snapshot build.Snapshot
	catalog  *Memory
}

var _ build.Repository = (*BuildMemory)(nil)
var _ catalog.Repository = (*BuildMemory)(nil)

func NewBuildMemory(source build.Snapshot) (*BuildMemory, error) {
	snapshot := build.Snapshot{
		Products: append([]build.Component{}, source.Products...), RAMModels: append([]build.RAMModel{}, source.RAMModels...),
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
			valid = specs.CPUSocket != "" && specs.RAMType == "" && specs.ModulesPerPackage == 0 && component.RAMModelID == ""
		case "motherboard":
			valid = specs.CPUSocket != "" && specs.RAMType != "" && specs.ModulesPerPackage == 0 && component.RAMModelID == ""
		case "ram":
			valid = specs.CPUSocket == "" && specs.RAMType != "" && (specs.ModulesPerPackage == 1 || specs.ModulesPerPackage == 2) && models[component.RAMModelID] == specs.RAMType
		default:
			valid = specs == (build.Specs{}) && component.RAMModelID == ""
		}
		if !valid {
			return nil, fmt.Errorf("%w: invalid compatibility metadata", build.ErrInvalidInput)
		}
		base = append(base, product)
	}
	baseRepository, err := NewMemory(base)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", build.ErrInvalidInput, err)
	}
	sort.Slice(snapshot.Products, func(i, j int) bool { return snapshot.Products[i].Product.ID < snapshot.Products[j].Product.ID })
	sort.Slice(snapshot.RAMModels, func(i, j int) bool { return snapshot.RAMModels[i].ID < snapshot.RAMModels[j].ID })
	return &BuildMemory{snapshot: snapshot, catalog: baseRepository}, nil
}

func (m *BuildMemory) ListActive(ctx context.Context, filter catalog.Filter) ([]catalog.Product, int, error) {
	return m.catalog.ListActive(ctx, filter)
}

func (m *BuildMemory) FindActiveByID(ctx context.Context, id string) (catalog.Product, error) {
	return m.catalog.FindActiveByID(ctx, id)
}

func (m *BuildMemory) LoadSnapshot(ctx context.Context) (build.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return build.Snapshot{}, err
	}
	return build.Snapshot{
		Products: append([]build.Component{}, m.snapshot.Products...), RAMModels: append([]build.RAMModel{}, m.snapshot.RAMModels...),
	}, nil
}
