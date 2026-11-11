package build

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	"specuse/backend/internal/catalog"
	"specuse/backend/internal/hardware"
)

const gpuID = "30000000-0000-0000-0000-000000000010"
const storageID = "30000000-0000-0000-0000-000000000011"
const usbID = "30000000-0000-0000-0000-000000000012"

func extendedSnapshot() Snapshot {
	s := testSnapshot()
	kind := "usb"
	s.Products = append(s.Products,
		Component{Product: catalog.Product{ID: gpuID, Name: "Test GPU", Category: "gpu", Brand: "Synthetic", PriceTHB: 6000, IsActive: true}},
		Component{Product: catalog.Product{ID: storageID, Name: "Test drive", Category: "storage", Brand: "Synthetic", PriceTHB: 1000, IsActive: true}},
		Component{Product: catalog.Product{ID: usbID, Name: "Test USB drive", Category: "storage", Brand: "Synthetic", PriceTHB: 500, IsActive: true}, Specs: Specs{Storage: &hardware.Storage{Host: &hardware.StorageHost{ConnectionType: &kind}}}},
	)
	return s
}

func TestExtendedSoftwareLimitsCountAllInstalledDevices(t *testing.T) {
	for _, tt := range []struct {
		name  string
		items []Item
		want  error
		price int64
	}{
		{"empty", nil, nil, 0},
		{"gpu zero purchase", []Item{{gpuID, 1, 1}}, nil, 0},
		{"gpu one purchase", []Item{{gpuID, 1, 0}}, nil, 6000},
		{"gpu two fully owned", []Item{{gpuID, 2, 2}}, ErrUnsupportedScope, 0},
		{"gpu max int", []Item{{gpuID, math.MaxInt64, math.MaxInt64}}, ErrUnsupportedScope, 0},
		{"storage one", []Item{{storageID, 1, 0}}, nil, 1000},
		{"storage eight owned", []Item{{storageID, 8, 8}}, nil, 0},
		{"owned subset not added twice", []Item{{storageID, 3, 2}}, nil, 1000},
		{"mixed eight includes unsupported owned", []Item{{storageID, 5, 4}, {usbID, 3, 2}}, nil, 1500},
		{"mixed nine all owned", []Item{{storageID, 5, 5}, {usbID, 4, 4}}, ErrUnsupportedScope, 0},
		{"unsupported nine", []Item{{usbID, 9, 0}}, ErrUnsupportedScope, 0},
		{"storage max int", []Item{{storageID, 1, 1}, {usbID, math.MaxInt64, math.MaxInt64}}, ErrUnsupportedScope, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stubRepository{snapshot: extendedSnapshot()}
			result, err := NewService(repo).ValidateExtended(context.Background(), tt.items, nil)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err=%v want=%v", err, tt.want)
			}
			if err == nil && (result.TotalPriceTHB != tt.price || result.Compatibility.Mode != GPUStorageMode || result.Compatibility.Status != "unknown" || result.Compatible) {
				t.Fatalf("unexpected result: %+v", result)
			}
			if repo.calls != 1 {
				t.Fatalf("expected one authoritative snapshot, got %d", repo.calls)
			}
		})
	}
}

func TestExtendedUnavailableReferencesWinBeforeLimitsAndPrices(t *testing.T) {
	missing := "30000000-0000-0000-0000-000000000099"
	for _, tt := range []struct {
		name   string
		items  []Item
		rams   []RAMItem
		change func(*Snapshot)
	}{
		{"missing after limit", []Item{{storageID, math.MaxInt64, 0}, {missing, 1, 0}}, nil, nil},
		{"missing before limit", []Item{{missing, 1, 0}, {storageID, 9, 9}}, nil, nil},
		{"missing RAM model", []Item{{storageID, 9, 9}}, []RAMItem{{missing, 1, 0}}, nil},
		{"inactive owned GPU", []Item{{storageID, 9, 9}, {gpuID, 1, 1}}, nil, func(s *Snapshot) { s.Products[4].Product.IsActive = false }},
		{"no active RAM packages", []Item{{storageID, 9, 9}}, []RAMItem{{modelID, 1, 0}}, func(s *Snapshot) { s.Products[2].Product.IsActive = false; s.Products[3].Product.IsActive = false }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := extendedSnapshot()
			if tt.change != nil {
				tt.change(&s)
			}
			_, err := NewService(&stubRepository{snapshot: s}).ValidateExtended(context.Background(), tt.items, tt.rams)
			if !errors.Is(err, ErrUnavailable) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestExtendedCPUAndOwnedRAMFailuresBeatUnknownCoverage(t *testing.T) {
	for _, tt := range []struct {
		name                     string
		cpuMismatch, ramMismatch bool
		want                     string
		issues                   int
	}{
		{"basic pass leaves unknown", false, false, "unknown", 0},
		{"CPU conflict", true, false, "failed", 1},
		{"owned RAM conflict", false, true, "failed", 1},
		{"both conflicts", true, true, "failed", 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := extendedSnapshot()
			if tt.cpuMismatch {
				s.Products[0].Specs.CPUSocket = "OTHER"
			}
			if tt.ramMismatch {
				s.RAMModels[0].RAMType = "DDR4"
			}
			r, err := NewService(&stubRepository{snapshot: s}).ValidateExtended(context.Background(), []Item{{cpuID, 1, 1}, {boardID, 1, 0}, {gpuID, 1, 1}, {storageID, 3, 2}}, []RAMItem{{modelID, 2, 2}})
			if err != nil || r.Compatibility.Status != tt.want || r.Compatible || len(r.Issues) != tt.issues || r.TotalPriceTHB != 4000 {
				t.Fatalf("result=%+v err=%v", r, err)
			}
			for _, c := range r.Compatibility.Checks {
				if c.Scope != "basic" && c.Status != "unknown" {
					t.Fatalf("stage B/C check executed: %+v", c)
				}
			}
		})
	}
}

func TestExtendedMissingAndUnsupportedAreDistinct(t *testing.T) {
	s := extendedSnapshot()
	r, err := NewService(&stubRepository{snapshot: s}).ValidateExtended(context.Background(), []Item{{usbID, 1, 0}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range r.Compatibility.Checks {
		if c.MissingFields == nil || c.UnsupportedFields == nil || c.ProductIDs == nil {
			t.Fatalf("arrays must never be null: %+v", c)
		}
		if !reflect.DeepEqual(c.MissingFields, uniqueRefs(c.MissingFields)) {
			t.Fatalf("unstable refs: %+v", c)
		}
		if c.Code == "STORAGE_MOTHERBOARD" {
			if c.ReasonCode != "UNSUPPORTED_DEVICE_CLASS" || len(c.UnsupportedFields) != 1 {
				t.Fatalf("known USB class: %+v", c)
			}
			for _, ref := range c.MissingFields {
				if ref.Path == "/specs/host/connection_type" {
					t.Fatal("known value reported missing")
				}
			}
			ref := c.UnsupportedFields[0]
			if ref.Location != "catalog" || ref.ProductID == nil || *ref.ProductID != usbID || ref.Category == nil || *ref.Category != "storage" {
				t.Fatalf("bad reference %+v", ref)
			}
		}
		if c.Code == "SYSTEM_PSU" {
			found := false
			for _, ref := range c.MissingFields {
				if ref.Location == "request" && ref.Path == "/items" && ref.ProductID == nil && ref.Category != nil && *ref.Category == "psu" {
					found = true
				}
			}
			if !found {
				t.Fatal("missing PSU must have a nullable product reference")
			}
		}
	}
}

func TestExtendedInputValidationAndOverflow(t *testing.T) {
	for _, items := range [][]Item{{{"bad", 1, 0}}, {{gpuID, 0, 0}}, {{gpuID, 1, -1}}, {{storageID, 1, 2}}, {{storageID, 1, 0}, {storageID, 1, 0}}} {
		repo := &stubRepository{}
		_, err := NewService(repo).ValidateExtended(context.Background(), items, nil)
		if !errors.Is(err, ErrInvalidInput) || repo.calls != 0 {
			t.Fatalf("input=%+v err=%v calls=%d", items, err, repo.calls)
		}
	}
	for _, rams := range [][]RAMItem{{{modelID, 0, 0}}, {{modelID, 1, 2}}, {{"bad", 1, 0}}, {{modelID, 1, 0}, {modelID, 1, 0}}} {
		repo := &stubRepository{}
		_, err := NewService(repo).ValidateExtended(context.Background(), nil, rams)
		if !errors.Is(err, ErrInvalidInput) || repo.calls != 0 {
			t.Fatalf("rams=%+v err=%v", rams, err)
		}
	}
	s := extendedSnapshot()
	s.Products[0].Product.PriceTHB = math.MaxInt64
	for _, items := range [][]Item{{{cpuID, 2, 0}}, {{cpuID, 1, 0}, {storageID, 1, 0}}} {
		if _, err := NewService(&stubRepository{snapshot: s}).ValidateExtended(context.Background(), items, nil); !errors.Is(err, ErrOverflow) {
			t.Fatalf("price overflow: %v", err)
		}
	}
	want := errors.New("repository error")
	if _, err := NewService(&stubRepository{err: want}).ValidateExtended(context.Background(), nil, nil); !errors.Is(err, want) {
		t.Fatalf("repository error masked: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewService(&stubRepository{}).ValidateExtended(ctx, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error %v", err)
	}
}

func TestExtendedDoesNotChangeLegacyScopeOrEmptyDraft(t *testing.T) {
	s := NewService(&stubRepository{snapshot: extendedSnapshot()})
	r, err := s.Validate(context.Background(), []Item{{gpuID, 2, 2}, {storageID, 9, 9}}, nil)
	if err != nil || !r.Compatible || r.Compatibility != nil {
		t.Fatalf("legacy changed %+v %v", r, err)
	}
	r, err = s.ValidateExtended(context.Background(), nil, nil)
	if err != nil || r.Compatibility.Status != "unknown" || r.Compatible || len(r.Compatibility.Checks) != 0 || r.Compatibility.Checks == nil {
		t.Fatalf("empty opt-in %+v %v", r, err)
	}
	r, err = s.ValidateExtended(context.Background(), nil, []RAMItem{{modelID, 2, 2}})
	if err != nil || r.TotalPriceTHB != 0 || r.Compatibility.Status != "unknown" {
		t.Fatalf("owned-only RAM %+v %v", r, err)
	}
}

func TestOverallStatusPriority(t *testing.T) {
	for _, tt := range []struct {
		statuses []string
		want     string
	}{
		{nil, "unknown"}, {[]string{"passed"}, "passed"}, {[]string{"unknown"}, "unknown"},
		{[]string{"passed", "unknown"}, "unknown"}, {[]string{"failed", "unknown"}, "failed"},
		{[]string{"unknown", "passed", "failed"}, "failed"}, {[]string{"passed", "passed"}, "passed"},
	} {
		checks := []Check{}
		for _, status := range tt.statuses {
			checks = append(checks, Check{Status: status})
		}
		if got := aggregate(checks); got != tt.want {
			t.Fatalf("%v => %s want %s", tt.statuses, got, tt.want)
		}
	}
}

func TestExtendedMissingBasicMetadataDoesNotBecomeAProvenFailure(t *testing.T) {
	s := extendedSnapshot()
	s.Products[0].Specs.CPUSocket = ""
	r, err := NewService(&stubRepository{snapshot: s}).ValidateExtended(context.Background(), []Item{{cpuID, 1, 1}, {boardID, 1, 0}}, nil)
	if err != nil || len(r.Issues) != 0 || r.Compatibility.Status != "unknown" {
		t.Fatalf("missing is not conflict: %+v %v", r, err)
	}
	basic := r.Compatibility.Checks[0]
	if basic.Code != "CPU_SOCKET" || basic.Status != "unknown" || basic.ReasonCode != "METADATA_MISSING" || len(basic.MissingFields) != 1 || basic.MissingFields[0].Path != "/specs/cpu_socket" {
		t.Fatalf("bad basic metadata coverage: %+v", basic)
	}
	s.RAMModels[0].RAMType = "DDR4"
	r, err = NewService(&stubRepository{snapshot: s}).ValidateExtended(context.Background(), []Item{{cpuID, 1, 1}, {boardID, 1, 0}}, []RAMItem{{modelID, 2, 2}})
	if err != nil || len(r.Issues) != 1 || r.Issues[0].Code != "RAM_TYPE_MISMATCH" || r.Compatibility.Status != "failed" {
		t.Fatalf("proven conflict must win over missing CPU data: %+v %v", r, err)
	}
}
