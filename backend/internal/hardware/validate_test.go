package hardware

import (
	"encoding/json"
	"math"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestNullableMetadataAndNumbers(t *testing.T) {
	if err := ValidateGPU(&GPU{}, false); err != nil {
		t.Fatal(err)
	}
	if err := ValidateStorage(&Storage{Power: &StoragePower{StartupPowerW: ptr(float64(0))}}, false); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMotherboard(&Motherboard{}, false); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		g    *GPU
		s    *Storage
	}{
		{"negative VRAM", &GPU{Display: &GPUDisplay{VRAMCapacityBytes: ptr(int64(-1))}}, nil},
		{"zero dimensions", &GPU{Dimensions: &GPUDimensions{LengthMM: ptr(float64(0))}}, nil},
		{"invalid connector", &GPU{Host: &GPUHost{Connector: ptr("pcie_x8")}}, nil},
		{"invalid lane count", &GPU{Host: &GPUHost{SupportedLinkModes: []LinkMode{{4, 3}}}}, nil},
		{"invalid generation", &GPU{Host: &GPUHost{SupportedLinkModes: []LinkMode{{0, 16}}}}, nil},
		{"invalid power count", &GPU{Power: &GPUPower{ConnectorOptions: []PowerOption{{OptionID: "test", CablePolicy: "unspecified", Connectors: []PowerConnector{{Type: "pcie_8pin", Count: 0}}}}}}, nil},
		{"invalid no-cable count", &GPU{Power: &GPUPower{ConnectorOptions: []PowerOption{{OptionID: "test", CablePolicy: "unspecified", Connectors: []PowerConnector{{Type: "none", Count: 1}}}}}}, nil},
		{"storage negative power", nil, &Storage{Power: &StoragePower{StartupPowerW: ptr(float64(-1))}}},
		{"storage invalid protocol", nil, &Storage{Host: &StorageHost{ConnectionType: ptr("nvme-guessed")}}},
		{"storage invalid key", nil, &Storage{Host: &StorageHost{M2Keying: ptr("x")}}},
		{"storage negative capacity", nil, &Storage{Display: &StorageDisplay{CapacityBytes: ptr(int64(-1))}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.g != nil && ValidateGPU(tt.g, false) == nil {
				t.Fatal("accepted bad GPU shape")
			}
			if tt.s != nil && ValidateStorage(tt.s, false) == nil {
				t.Fatal("accepted bad storage shape")
			}
		})
	}
	if _, err := Clone(&GPU{Power: &GPUPower{BoardPowerW: ptr(math.Inf(1))}}); err == nil {
		t.Fatal("nonfinite metadata should fail serialization")
	}
}

func TestProvenanceAndProfileShapeValidation(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
	}{
		{"bad URL", `{"sources":[{"source_id":"x","manufacturer":"m","title":"t","url":"http://example.invalid","section":"s","checked_at":"2026-11-11T00:00:00Z"}]}`},
		{"unresolved evidence", `{"evidence":[{"paths":["/host"],"source_ids":["missing"]}]}`},
		{"empty evidence", `{"evidence":[{"paths":[],"source_ids":[]}]}`},
		{"CPU wildcard", `{"connectivity_profiles":[{"profile_id":"x","cpu_product_ids":[]}]}`},
		{"bad CPU UUID", `{"connectivity_profiles":[{"profile_id":"x","cpu_product_ids":["bad"]}]}`},
		{"duplicate endpoints", `{"connectivity_profiles":[{"profile_id":"x","pcie_slots":[{"slot_id":"same"}],"storage_ports":[{"port_id":"same"}]}]}`},
		{"invalid storage protocol", `{"connectivity_profiles":[{"profile_id":"x","storage_ports":[{"port_id":"S1","supported_connection_types":["invented"]}]}]}`},
		{"dangling rule endpoint", `{"sources":[{"source_id":"s","manufacturer":"SYNTHETIC","title":"test","url":"https://example.invalid","section":"test","checked_at":"2026-11-11T00:00:00Z"}],"connectivity_profiles":[{"profile_id":"x","resource_rules":[{"rule_id":"r","when_all_occupied":["missing"],"effect":"disable","affected_endpoint_ids":["missing"],"allowed_link_modes":null,"required_context":[],"source_ids":["s"]}]}]}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var m Motherboard
			if err := json.Unmarshal([]byte(tt.body), &m); err != nil {
				t.Fatal(err)
			}
			if err := ValidateMotherboard(&m, true); err == nil {
				t.Fatal("invalid stored metadata accepted")
			}
		})
	}
}
