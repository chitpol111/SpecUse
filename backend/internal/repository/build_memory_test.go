package repository

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
)

func buildSnapshot() build.Snapshot {
	model := "10000000-0000-0000-0000-000000000001"
	return build.Snapshot{
		Products: []build.Component{
			{Product: catalog.Product{ID: "20000000-0000-0000-0000-000000000002", Name: "RAM", Category: "ram", Brand: "Example", PriceTHB: 1000, IsActive: true},
				Specs: build.Specs{RAMType: "DDR5", ModulesPerPackage: 2}, RAMModelID: model},
			{Product: catalog.Product{ID: "20000000-0000-0000-0000-000000000001", Name: "CPU", Category: "cpu", Brand: "Example", PriceTHB: 5000, IsActive: true},
				Specs: build.Specs{CPUSocket: "AM5"}},
		},
		RAMModels: []build.RAMModel{{ID: model, RAMType: "DDR5"}},
	}
}

func TestCatalogSpecCopiesCannotChangeBuildValidation(t *testing.T) {
	source := buildSnapshot()
	board := source.Products[1]
	board.Product.ID = "20000000-0000-0000-0000-000000000003"
	board.Product.Category = "motherboard"
	board.Specs.RAMType = "DDR5"
	source.Products = append(source.Products, board)
	repo, err := NewBuildMemory(source)
	if err != nil {
		t.Fatal(err)
	}
	cpu, err := repo.FindActiveByID(context.Background(), source.Products[1].Product.ID)
	if err != nil || cpu.Specs == nil {
		t.Fatalf("CPU=%+v err=%v", cpu, err)
	}
	cpu.Specs.CPUSocket = "modified"
	ram, err := repo.FindActiveByID(context.Background(), source.Products[0].Product.ID)
	if err != nil || ram.Specs == nil {
		t.Fatalf("RAM=%+v err=%v", ram, err)
	}
	ram.Specs.RAMType, ram.Specs.ModulesPerPackage = "modified", 1
	items, _, err := repo.ListActive(context.Background(), catalog.Filter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Specs.CPUSocket != "AM5" || items[1].Specs.RAMType != "DDR5" || items[1].Specs.ModulesPerPackage != 2 {
		t.Fatalf("detail response mutated repository: %+v", items)
	}
	items[0].Specs.CPUSocket = "also modified"
	result, err := build.NewService(repo).Validate(context.Background(),
		[]build.Item{{ProductID: source.Products[1].Product.ID, Quantity: 1, OwnedQuantity: 1}, {ProductID: board.Product.ID, Quantity: 1, OwnedQuantity: 1}},
		[]build.RAMItem{{RAMModelID: source.RAMModels[0].ID, RequiredModules: 2, OwnedModules: 0}})
	if err != nil || !result.Compatible || result.TotalPriceTHB != 1000 || result.RAMSelections[0].Packages[0].ModulesPerPackage != 2 {
		t.Fatalf("public specs changed validation: %+v %v", result, err)
	}
}

func TestCatalogOmitsUnagreedSpecsForOtherCategories(t *testing.T) {
	for _, category := range []string{"gpu", "storage", "psu", "case", "cooler"} {
		t.Run(category, func(t *testing.T) {
			source := buildSnapshot()
			source.Products[1].Product.Category = category
			source.Products[1].Specs = build.Specs{}
			repo, err := NewBuildMemory(source)
			if err != nil {
				t.Fatal(err)
			}
			product, err := repo.FindActiveByID(context.Background(), source.Products[1].Product.ID)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(product)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			if len(fields) != 6 || product.Specs != nil || product.RAMModelID != "" {
				t.Fatalf("unagreed fields for %s: %s", category, encoded)
			}
		})
	}
}

func TestBuildMemoryValidatesMetadataAndRelationships(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*build.Snapshot)
	}{
		{"uppercase category", func(s *build.Snapshot) { s.Products[0].Product.Category = "RAM" }},
		{"bad product UUID", func(s *build.Snapshot) { s.Products[0].Product.ID = "bad" }},
		{"negative price", func(s *build.Snapshot) { s.Products[0].Product.PriceTHB = -1 }},
		{"duplicate product", func(s *build.Snapshot) { s.Products = append(s.Products, s.Products[0]) }},
		{"missing socket", func(s *build.Snapshot) { s.Products[1].Specs.CPUSocket = " " }},
		{"wrong category specs", func(s *build.Snapshot) { s.Products[1].Specs.RAMType = "DDR5" }},
		{"missing motherboard RAM type", func(s *build.Snapshot) { s.Products[1].Product.Category = "motherboard" }},
		{"bad package size", func(s *build.Snapshot) { s.Products[0].Specs.ModulesPerPackage = 4 }},
		{"missing model", func(s *build.Snapshot) { s.RAMModels = nil }},
		{"model/type mismatch", func(s *build.Snapshot) { s.RAMModels[0].RAMType = "DDR4" }},
		{"duplicate model", func(s *build.Snapshot) { s.RAMModels = append(s.RAMModels, s.RAMModels[0]) }},
		{"invalid model UUID", func(s *build.Snapshot) { s.RAMModels[0].ID = "bad" }},
		{"empty model RAM type", func(s *build.Snapshot) { s.RAMModels[0].RAMType = "" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := buildSnapshot()
			tt.change(&snapshot)
			if _, err := NewBuildMemory(snapshot); !errors.Is(err, build.ErrInvalidInput) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestBuildMemoryCopiesSnapshotsAndPreservesInactiveStatus(t *testing.T) {
	source := buildSnapshot()
	source.Products[0].Product.IsActive = false
	repo, err := NewBuildMemory(source)
	if err != nil {
		t.Fatal(err)
	}
	source.Products[0].Specs.RAMType = "modified"
	source.RAMModels[0].RAMType = "modified"
	first, err := repo.LoadSnapshot(context.Background())
	if err != nil || first.Products[0].Product.Category != "cpu" || first.Products[1].Product.IsActive || first.Products[1].Specs.RAMType != "DDR5" {
		t.Fatalf("snapshot=%+v err=%v", first, err)
	}
	first.Products[0].Product.PriceTHB = 0
	first.RAMModels[0].RAMType = "modified"
	second, _ := repo.LoadSnapshot(context.Background())
	if second.Products[0].Product.PriceTHB != 5000 || second.RAMModels[0].RAMType != "DDR5" {
		t.Fatalf("snapshot mutation leaked: %+v", second)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repo.LoadSnapshot(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestBuildMemorySupportsAgreedCategoriesWithoutGuessingOtherSpecs(t *testing.T) {
	snapshot := buildSnapshot()
	snapshot.Products[1].Product.Category = "motherboard"
	snapshot.Products[1].Specs.RAMType = "DDR5"
	if _, err := NewBuildMemory(snapshot); err != nil {
		t.Fatalf("valid motherboard: %v", err)
	}
	snapshot.Products[1].Product.Category = "gpu"
	snapshot.Products[1].Specs = build.Specs{}
	if _, err := NewBuildMemory(snapshot); err != nil {
		t.Fatalf("base-only GPU: %v", err)
	}
	snapshot.Products[1].Specs.RAMType = "DDR5"
	if _, err := NewBuildMemory(snapshot); !errors.Is(err, build.ErrInvalidInput) {
		t.Fatalf("unagreed GPU specs: %v", err)
	}
}

func TestBuildRepositoryAndServiceIntegrate(t *testing.T) {
	snapshot := buildSnapshot()
	repo, err := NewBuildMemory(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	result, err := build.NewService(repo).Validate(context.Background(),
		[]build.Item{{ProductID: snapshot.Products[1].Product.ID, Quantity: 1, OwnedQuantity: 1}},
		[]build.RAMItem{{RAMModelID: snapshot.RAMModels[0].ID, RequiredModules: 3, OwnedModules: 0}})
	if err != nil || !result.Compatible || result.TotalPriceTHB != 2000 || result.RAMSelections[0].ExcessModules != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestBuildMemoryCatalogAndValidationUseSameData(t *testing.T) {
	source := buildSnapshot()
	repo, err := NewBuildMemory(source)
	if err != nil {
		t.Fatal(err)
	}
	product, err := repo.FindActiveByID(context.Background(), source.Products[0].Product.ID)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := repo.LoadSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var snapshotPrice int64
	for _, component := range snapshot.Products {
		if component.Product.ID == product.ID {
			snapshotPrice = component.Product.PriceTHB
		}
	}
	result, err := build.NewService(repo).Validate(context.Background(), nil,
		[]build.RAMItem{{RAMModelID: source.RAMModels[0].ID, RequiredModules: 2, OwnedModules: 0}})
	if err != nil || product.PriceTHB != snapshotPrice || result.TotalPriceTHB != product.PriceTHB {
		t.Fatalf("catalog=%+v snapshot price=%d result=%+v err=%v", product, snapshotPrice, result, err)
	}
	items, total, err := repo.ListActive(context.Background(), catalog.Filter{Category: "ram", Page: 1, PageSize: 20})
	if err != nil || total != 1 || len(items) != 1 || !reflect.DeepEqual(items[0], product) {
		t.Fatalf("catalog filter: %+v %d %v", items, total, err)
	}
	source.Products[0].Product.IsActive = false
	inactiveRepo, err := NewBuildMemory(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inactiveRepo.FindActiveByID(context.Background(), product.ID); !errors.Is(err, catalog.ErrNotFound) {
		t.Fatalf("inactive product leaked: %v", err)
	}
}
