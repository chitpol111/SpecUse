package build

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"testing"

	"specuse/backend/internal/catalog"
	"specuse/backend/internal/hardware"
)

const secondSATAID = "30000000-0000-0000-0000-000000000013"

func b1ptr[T any](v T) *T { return &v }
func syntheticEvidence(path string) hardware.Provenance {
	return hardware.Provenance{
		Sources:  []hardware.Source{{SourceID: "synthetic-b1", Manufacturer: "SYNTHETIC FIXTURE ONLY", Title: "Artificial B1 test model; not manufacturer evidence", URL: "https://fixtures.specuse.invalid/b1", Section: "unit test only", CheckedAt: "2026-11-11T00:00:00Z"}},
		Evidence: []hardware.Evidence{{Paths: []string{path}, SourceIDs: []string{"synthetic-b1"}}},
	}
}
func syntheticSATA() *hardware.Storage {
	return &hardware.Storage{Host: &hardware.StorageHost{ConnectionType: b1ptr("sata"), Connector: b1ptr("sata_data"), FormFactor: b1ptr("2_5_inch")}, Provenance: syntheticEvidence("/host")}
}
func b1Snapshot(portCount int) Snapshot {
	s := extendedSnapshot()
	ports := []hardware.StoragePort{}
	for i := 0; i < portCount; i++ {
		ports = append(ports, hardware.StoragePort{PortID: fmt.Sprintf("SATA_%d", i+1), Connector: b1ptr("sata_data"), SupportedConnectionTypes: []string{"sata"}})
	}
	s.Products[1].Specs.Motherboard = &hardware.Motherboard{Provenance: syntheticEvidence("/connectivity_profiles"), ConnectivityProfiles: []hardware.IOProfile{{ProfileID: "synthetic-cpu", CPUProductIDs: []string{cpuID}, RequiresFirmwareConfirmation: b1ptr(false), StoragePorts: ports, ResourceRules: []hardware.ResourceRule{}}}}
	s.Products[5].Specs.Storage = syntheticSATA()
	s.Products = append(s.Products, Component{Product: catalog.Product{ID: secondSATAID, Name: "SYNTHETIC second SATA SKU", Category: "storage", Brand: "SYNTHETIC", PriceTHB: 1500, IsActive: true}, Specs: Specs{Storage: syntheticSATA()}})
	return s
}
func b1Items(q, owned int64) []Item {
	return []Item{{cpuID, 1, 1}, {boardID, 1, 1}, {storageID, q, owned}}
}
func runB1(t *testing.T, s Snapshot, items []Item) Result {
	t.Helper()
	repo := &stubRepository{snapshot: s}
	r, err := NewService(repo).ValidateExtended(context.Background(), items, nil)
	if err != nil {
		t.Fatal(err)
	}
	if repo.calls != 1 {
		t.Fatal("must read one snapshot")
	}
	return r
}
func b1Check(t *testing.T, r Result, code string) Check {
	t.Helper()
	for _, c := range r.Compatibility.Checks {
		if c.Code == code {
			return c
		}
	}
	t.Fatalf("missing %s", code)
	return Check{}
}
func TestB1CountsOwnedAndKeepsPurchasePricing(t *testing.T) {
	r := runB1(t, b1Snapshot(3), b1Items(3, 2))
	for _, code := range []string{"STORAGE_MOTHERBOARD", "MOTHERBOARD_RESOURCES"} {
		c := b1Check(t, r, code)
		if c.Status != "passed" || len(c.MissingFields) != 0 || c.MissingFields == nil || c.UnsupportedFields == nil {
			t.Fatalf("bad check %+v", c)
		}
	}
	if r.TotalPriceTHB != 1000 || r.Items[2].PurchaseQuantity != 1 || r.Compatibility.Status != "unknown" || r.Compatible {
		t.Fatalf("bad pricing/overall %+v", r)
	}
	for _, code := range []string{"SYSTEM_PSU", "SYSTEM_CASE", "STORAGE_PSU", "STORAGE_CASE"} {
		if b1Check(t, r, code).Status != "unknown" {
			t.Fatal("PSU/Case evaluated")
		}
	}
	if r = runB1(t, b1Snapshot(2), b1Items(3, 2)); r.Compatibility.Status != "failed" || r.TotalPriceTHB != 1000 {
		t.Fatal("owned must consume ports")
	}
	s := b1Snapshot(3)
	s.Products[5].Product.PriceTHB = 1200
	if r = runB1(t, s, b1Items(3, 2)); r.TotalPriceTHB != 1200 {
		t.Fatal("price must come from catalog")
	}
}
func TestB1PortBoundaries(t *testing.T) {
	for _, tt := range []struct {
		name        string
		ports       int
		q, owned    int64
		want, issue string
	}{
		{"one", 1, 1, 0, "passed", ""}, {"N", 3, 3, 0, "passed", ""}, {"N owned", 3, 3, 3, "passed", ""},
		{"N plus one", 3, 4, 0, "failed", "MOTHERBOARD_RESOURCE_CONFLICT"}, {"confirmed none", 0, 1, 1, "failed", "STORAGE_INTERFACE_MISMATCH"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := runB1(t, b1Snapshot(tt.ports), b1Items(tt.q, tt.owned))
			for _, code := range []string{"STORAGE_MOTHERBOARD", "MOTHERBOARD_RESOURCES"} {
				if b1Check(t, r, code).Status != tt.want {
					t.Fatalf("wrong %s: %+v", code, r)
				}
			}
			if tt.issue != "" {
				if r.Compatibility.Status != "failed" || len(r.Issues) != 1 || r.Issues[0].Code != tt.issue {
					t.Fatal("failed must beat unknown")
				}
			} else if r.Compatibility.Status != "unknown" {
				t.Fatal("partial passed is not overall passed")
			}
		})
	}
}
func TestB1MultipleSKUsAndDeterministicDistinctPortAllocation(t *testing.T) {
	s := b1Snapshot(3)
	items := append(b1Items(2, 1), Item{secondSATAID, 1, 0})
	first := runB1(t, s, items)
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	p := s.Products[1].Specs.Motherboard.ConnectivityProfiles[0].StoragePorts
	p[0], p[2] = p[2], p[0]
	second := runB1(t, s, items)
	if first.TotalPriceTHB != 2500 || second.TotalPriceTHB != 2500 || !reflect.DeepEqual(first.Compatibility, second.Compatibility) || !reflect.DeepEqual(first.Issues, second.Issues) {
		t.Fatal("order affected result")
	}
	items[0].Quantity = 2
	if runB1(t, s, items).Compatibility.Status != "failed" {
		t.Fatal("SKUs must share port pool")
	}
	rows := []PricedItem{{Item: Item{secondSATAID, 1, 0}}, {Item: Item{storageID, 2, 1}}}
	witness, ok := allocateSATA(rows, []string{"SATA_3", "SATA_1", "SATA_2"})
	if !ok || len(witness) != 3 {
		t.Fatal("missing allocation")
	}
	seen := map[string]bool{}
	for _, a := range witness {
		if seen[a.PortID] {
			t.Fatal("port reused")
		}
		seen[a.PortID] = true
	}
	rows[0], rows[1] = rows[1], rows[0]
	other, ok := allocateSATA(rows, []string{"SATA_2", "SATA_3", "SATA_1"})
	if !ok || !reflect.DeepEqual(witness, other) {
		t.Fatal("allocation must use UUID/instance and port-ID order")
	}
	for _, ports := range [][]string{{"S", "S"}, {""}} {
		if _, ok := allocateSATA(rows, ports); ok {
			t.Fatal("invalid ports accepted")
		}
	}
	if _, ok := allocateSATA([]PricedItem{{Item: Item{storageID, math.MaxInt64, 0}}}, []string{"S"}); ok {
		t.Fatal("huge quantity accepted")
	}
}
func TestB1MissingDataAndEvidenceAreUnknown(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*Snapshot)
	}{
		{"ports null", func(s *Snapshot) { s.Products[1].Specs.Motherboard.ConnectivityProfiles[0].StoragePorts = nil }},
		{"profiles null", func(s *Snapshot) { s.Products[1].Specs.Motherboard.ConnectivityProfiles = nil }},
		{"profiles empty", func(s *Snapshot) { s.Products[1].Specs.Motherboard.ConnectivityProfiles = []hardware.IOProfile{} }},
		{"CPU mismatch", func(s *Snapshot) {
			s.Products[1].Specs.Motherboard.ConnectivityProfiles[0].CPUProductIDs = []string{gpuID}
		}},
		{"CPU applicability unknown", func(s *Snapshot) { s.Products[1].Specs.Motherboard.ConnectivityProfiles[0].CPUProductIDs = nil }},
		{"ambiguous profile", func(s *Snapshot) {
			m := s.Products[1].Specs.Motherboard
			m.ConnectivityProfiles = append(m.ConnectivityProfiles, m.ConnectivityProfiles[0])
		}},
		{"possibly applicable profile", func(s *Snapshot) {
			m := s.Products[1].Specs.Motherboard
			m.ConnectivityProfiles = append(m.ConnectivityProfiles, hardware.IOProfile{ProfileID: "unknown"})
		}},
		{"board sources missing", func(s *Snapshot) { s.Products[1].Specs.Motherboard.Sources = nil }},
		{"board evidence missing", func(s *Snapshot) { s.Products[1].Specs.Motherboard.Evidence = nil }},
		{"board display evidence", func(s *Snapshot) { s.Products[1].Specs.Motherboard.Evidence[0].Paths = []string{"/display"} }},
		{"drive evidence missing", func(s *Snapshot) { s.Products[5].Specs.Storage.Evidence = nil }},
		{"unresolved evidence", func(s *Snapshot) { s.Products[5].Specs.Storage.Evidence[0].SourceIDs = []string{"unknown"} }},
		{"incomplete host evidence", func(s *Snapshot) { s.Products[5].Specs.Storage.Evidence[0].Paths = []string{"/host/connector"} }},
		{"host missing", func(s *Snapshot) { s.Products[5].Specs.Storage.Host = nil }},
		{"connector missing", func(s *Snapshot) { s.Products[5].Specs.Storage.Host.Connector = nil }},
		{"interface missing", func(s *Snapshot) { s.Products[5].Specs.Storage.Host.ConnectionType = nil }},
		{"form factor missing", func(s *Snapshot) { s.Products[5].Specs.Storage.Host.FormFactor = nil }},
		{"port connector missing", func(s *Snapshot) {
			s.Products[1].Specs.Motherboard.ConnectivityProfiles[0].StoragePorts[0].Connector = nil
		}},
		{"port interface missing", func(s *Snapshot) {
			s.Products[1].Specs.Motherboard.ConnectivityProfiles[0].StoragePorts[0].SupportedConnectionTypes = nil
		}},
		{"rules missing", func(s *Snapshot) { s.Products[1].Specs.Motherboard.ConnectivityProfiles[0].ResourceRules = nil }},
		{"firmware metadata missing", func(s *Snapshot) {
			s.Products[1].Specs.Motherboard.ConnectivityProfiles[0].RequiresFirmwareConfirmation = nil
		}},
		{"duplicate port IDs", func(s *Snapshot) {
			p := s.Products[1].Specs.Motherboard.ConnectivityProfiles[0].StoragePorts
			p[1].PortID = p[0].PortID
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := b1Snapshot(3)
			tt.change(&s)
			r := runB1(t, s, b1Items(3, 2))
			for _, code := range []string{"STORAGE_MOTHERBOARD", "MOTHERBOARD_RESOURCES"} {
				c := b1Check(t, r, code)
				if c.Status != "unknown" || c.MissingFields == nil || c.UnsupportedFields == nil {
					t.Fatalf("incomplete data produced verdict %+v", c)
				}
			}
			if len(r.Issues) != 0 || r.Compatibility.Status != "unknown" {
				t.Fatal("missing data is not incompatibility")
			}
		})
	}
	s := b1Snapshot(0)
	s.Products[1].Specs.Motherboard.Evidence = nil
	if runB1(t, s, b1Items(1, 0)).Compatibility.Status != "unknown" {
		t.Fatal("unverified empty ports must not fail")
	}
}
func TestB1UnimplementedSelectionsAndConstraintsAreUnknown(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*Snapshot, *[]Item)
	}{
		{"CPU absent", func(s *Snapshot, i *[]Item) { *i = (*i)[1:] }},
		{"board absent", func(s *Snapshot, i *[]Item) { *i = append((*i)[:1], (*i)[2:]...) }},
		{"CPU quantity two", func(s *Snapshot, i *[]Item) { (*i)[0].Quantity = 2 }},
		{"board quantity two", func(s *Snapshot, i *[]Item) { (*i)[1].Quantity = 2 }},
		{"GPU present", func(s *Snapshot, i *[]Item) { *i = append(*i, Item{gpuID, 1, 1}) }},
		{"M2 present", func(s *Snapshot, i *[]Item) {
			d := s.Products[len(s.Products)-1].Specs.Storage
			d.Host = &hardware.StorageHost{ConnectionType: b1ptr("pcie_nvme"), Connector: b1ptr("m2"), FormFactor: b1ptr("m2")}
			*i = append(*i, Item{secondSATAID, 1, 0})
		}},
		{"sharing", func(s *Snapshot, i *[]Item) {
			s.Products[1].Specs.Motherboard.ConnectivityProfiles[0].ResourceRules = []hardware.ResourceRule{{RuleID: "synthetic", Effect: "disable", WhenAllOccupied: []string{"SATA_1"}, AffectedEndpointIDs: []string{"SATA_2"}, SourceIDs: []string{"synthetic-b1"}, RequiredContext: []string{}}}
		}},
		{"BIOS", func(s *Snapshot, i *[]Item) {
			s.Products[1].Specs.Motherboard.ConnectivityProfiles[0].RequiresFirmwareConfirmation = b1ptr(true)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := b1Snapshot(3)
			items := b1Items(3, 2)
			tt.change(&s, &items)
			r := runB1(t, s, items)
			if b1Check(t, r, "STORAGE_MOTHERBOARD").Status != "unknown" || b1Check(t, r, "MOTHERBOARD_RESOURCES").Status != "unknown" || len(r.Issues) != 0 {
				t.Fatal("out-of-scope check evaluated")
			}
		})
	}
}
func TestB1UnusedMetadataAndM2PortsDoNotInflateCapacity(t *testing.T) {
	s := b1Snapshot(3)
	m := s.Products[1].Specs.Motherboard
	m.ConnectivityProfiles[0].StoragePorts = append(m.ConnectivityProfiles[0].StoragePorts, hardware.StoragePort{PortID: "M2_1", Connector: b1ptr("m2")})
	if b1Check(t, runB1(t, s, b1Items(3, 2)), "STORAGE_MOTHERBOARD").Status != "passed" {
		t.Fatal("inapplicable fields must not block SATA")
	}
	if runB1(t, s, b1Items(4, 2)).Compatibility.Status != "failed" {
		t.Fatal("M2 port counted as SATA data port")
	}
}
func TestB1LegacyAndProvenFailurePriority(t *testing.T) {
	s := b1Snapshot(3)
	r, err := NewService(&stubRepository{snapshot: s}).Validate(context.Background(), b1Items(4, 0), nil)
	if err != nil || !r.Compatible || r.Compatibility != nil || len(r.Issues) != 0 || r.TotalPriceTHB != 4000 {
		t.Fatal("legacy changed")
	}
	s.Products[0].Specs.CPUSocket = "OTHER"
	s.Products[1].Specs.Motherboard.Evidence = nil
	r = runB1(t, s, b1Items(1, 0))
	if r.Compatibility.Status != "failed" || len(r.Issues) != 1 || r.Issues[0].Code != "CPU_SOCKET_MISMATCH" || b1Check(t, r, "STORAGE_MOTHERBOARD").Status != "unknown" {
		t.Fatal("proven failed must win over unknown")
	}
}

func TestB1DirectSATAConnectorAndInterface(t *testing.T) {
	for _, tt := range []struct {
		name      string
		connector *string
		modes     []string
		want      string
	}{
		{"direct SATA", b1ptr("sata_data"), []string{"sata"}, "passed"},
		{"only M2 SATA port", b1ptr("m2"), []string{"sata"}, "failed"},
		{"confirmed no supported interface", b1ptr("sata_data"), []string{}, "failed"},
		{"interface not known", b1ptr("sata_data"), nil, "unknown"},
		{"connector not known", nil, []string{"sata"}, "unknown"},
		{"unrecognized port class", b1ptr("other"), []string{"sata"}, "unknown"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := b1Snapshot(1)
			p := &s.Products[1].Specs.Motherboard.ConnectivityProfiles[0].StoragePorts[0]
			p.Connector, p.SupportedConnectionTypes = tt.connector, tt.modes
			r := runB1(t, s, b1Items(1, 0))
			if c := b1Check(t, r, "STORAGE_MOTHERBOARD"); c.Status != tt.want {
				t.Fatalf("connector/interface: %+v", c)
			}
		})
	}
}
