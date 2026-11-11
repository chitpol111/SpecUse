package repository

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/hardware"
)

func gpuSnapshot(t *testing.T) build.Snapshot {
	t.Helper()
	var p catalog.ProductDetails
	if err := json.Unmarshal([]byte(`{"id":"40000000-0000-4000-8000-000000000001","name":"Synthetic GPU","category":"gpu","brand":"Synthetic","price_thb":6000,"is_active":true,"specs":{"display":{"chipset_name":"Synthetic","vram_capacity_bytes":8000000000},"host":{"connector":"pcie_x16","supported_link_modes":[{"pcie_generation":4,"lanes":16}]},"power":{"board_power_w":120,"min_system_power_w":500,"connector_options":[]},"dimensions":{"length_mm":240,"height_mm":115,"thickness_mm":40,"occupied_slots":2},"sources":[],"evidence":[]}}`), &p); err != nil {
		t.Fatal(err)
	}
	return build.Snapshot{Products: []build.Component{{Product: p.Product, Specs: build.Specs{GPU: p.Specs.GPU}}}}
}

func TestNestedMetadataCannotMutateSharedSnapshot(t *testing.T) {
	source := gpuSnapshot(t)
	id := source.Products[0].Product.ID
	repo, err := NewBuildMemory(source)
	if err != nil {
		t.Fatal(err)
	}
	*source.Products[0].Specs.GPU.Dimensions.LengthMM = 1
	source.Products[0].Specs.GPU.Host.SupportedLinkModes[0].Lanes = 1
	first, err := repo.FindActiveByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if *first.Specs.GPU.Dimensions.LengthMM != 240 || first.Specs.GPU.Host.SupportedLinkModes[0].Lanes != 16 {
		t.Fatal("constructor retained input aliases")
	}
	*first.Specs.GPU.Dimensions.LengthMM = 2
	first.Specs.GPU.Host.SupportedLinkModes[0].Lanes = 2
	snapshot, err := repo.LoadSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	*snapshot.Products[0].Specs.GPU.Dimensions.LengthMM = 3
	snapshot.Products[0].Specs.GPU.Host.SupportedLinkModes[0].Lanes = 4
	items, _, err := repo.ListActive(context.Background(), catalog.Filter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	second, _ := repo.FindActiveByID(context.Background(), id)
	if !reflect.DeepEqual(items[0], second) || *second.Specs.GPU.Dimensions.LengthMM != 240 || second.Specs.GPU.Host.SupportedLinkModes[0].Lanes != 16 {
		t.Fatal("response/snapshot metadata mutation leaked")
	}
	result, err := build.NewService(repo).ValidateExtended(context.Background(), []build.Item{{ProductID: id, Quantity: 1, OwnedQuantity: 0}}, nil)
	if err != nil || result.TotalPriceTHB != second.PriceTHB {
		t.Fatalf("catalog/validate prices drifted: %+v %v", result, err)
	}
}

func TestNullableMetadataKeepsNullDistinctFromReviewedEmpty(t *testing.T) {
	for _, category := range []string{"gpu", "storage"} {
		t.Run(category, func(t *testing.T) {
			s := gpuSnapshot(t)
			s.Products[0].Product.Category = category
			s.Products[0].Specs = build.Specs{}
			repo, err := NewBuildMemory(s)
			if err != nil {
				t.Fatal(err)
			}
			p, _ := repo.FindActiveByID(context.Background(), s.Products[0].Product.ID)
			data, _ := json.Marshal(p)
			var fields map[string]any
			_ = json.Unmarshal(data, &fields)
			specs := fields["specs"].(map[string]any)
			for _, name := range []string{"display", "host", "power", "dimensions", "sources", "evidence"} {
				if val, ok := specs[name]; !ok || val != nil {
					t.Fatalf("missing %s must be null: %s", name, data)
				}
			}
		})
	}
	s := gpuSnapshot(t)
	repo, err := NewBuildMemory(s)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := repo.FindActiveByID(context.Background(), s.Products[0].Product.ID)
	data, _ := json.Marshal(p)
	var copy catalog.ProductDetails
	if err := json.Unmarshal(data, &copy); err != nil {
		t.Fatal(err)
	}
	if copy.Specs.GPU.Sources == nil || copy.Specs.GPU.Evidence == nil || copy.Specs.GPU.Power.ConnectorOptions == nil {
		t.Fatal("reviewed [] changed to null")
	}
}

func TestSyntheticProvenanceRequiresLocalRepository(t *testing.T) {
	s := gpuSnapshot(t)
	s.Products[0].Specs.GPU.Sources = []hardware.Source{{SourceID: "synthetic-v1", Manufacturer: "SYNTHETIC", Title: "Test only", URL: "https://fixtures.specuse.invalid/test", Section: "test", CheckedAt: "2026-11-11T00:00:00Z"}}
	s.Products[0].Specs.GPU.Evidence = []hardware.Evidence{{Paths: []string{"/host"}, SourceIDs: []string{"synthetic-v1"}}}
	if _, err := NewBuildMemory(s); !errors.Is(err, build.ErrInvalidInput) {
		t.Fatalf("production accepted synthetic: %v", err)
	}
	if _, err := NewLocalBuildMemory(s); err != nil {
		t.Fatalf("explicit local rejected: %v", err)
	}
	s.Products[0].Specs.GPU.Host.Connector = new(string)
	*s.Products[0].Specs.GPU.Host.Connector = "invented"
	if _, err := NewLocalBuildMemory(s); !errors.Is(err, build.ErrInvalidInput) {
		t.Fatalf("bad enum accepted: %v", err)
	}
}
