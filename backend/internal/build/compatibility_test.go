package build

import "testing"

func TestCompatibilityChecksOnlyAgreedFields(t *testing.T) {
	for _, tt := range []struct {
		name                                         string
		cpuSocket, boardSocket, modelType, boardType string
		wantCPU, wantRAM                             bool
	}{
		{"match", "AM5", "AM5", "DDR5", "DDR5", false, false},
		{"socket mismatch", "LGA1700", "AM5", "DDR5", "DDR5", true, false},
		{"RAM mismatch", "AM5", "AM5", "DDR4", "DDR5", false, true},
		{"both mismatch", "LGA1700", "AM5", "DDR4", "DDR5", true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := testSnapshot()
			snapshot.Products[0].Specs.CPUSocket = tt.cpuSocket
			snapshot.Products[1].Specs.CPUSocket = tt.boardSocket
			snapshot.Products[1].Specs.RAMType = tt.boardType
			snapshot.RAMModels[0].RAMType = tt.modelType
			issues := CheckCompatibility(snapshot.Products[:2], snapshot.RAMModels)
			foundCPU, foundRAM := false, false
			for _, issue := range issues {
				if issue.Code == "CPU_SOCKET_MISMATCH" {
					foundCPU = true
				}
				if issue.Code == "RAM_TYPE_MISMATCH" {
					foundRAM = true
				}
			}
			if foundCPU != tt.wantCPU || foundRAM != tt.wantRAM {
				t.Fatalf("issues=%+v", issues)
			}
		})
	}
}

func TestCompatibilityMissingMetadataDoesNotSilentlyPass(t *testing.T) {
	snapshot := testSnapshot()
	snapshot.Products[0].Specs.CPUSocket = ""
	snapshot.Products[1].Specs = Specs{}
	snapshot.RAMModels[0].RAMType = ""
	issues := CheckCompatibility(snapshot.Products[:2], snapshot.RAMModels)
	if len(issues) != 4 {
		t.Fatalf("expected four missing metadata issues: %+v", issues)
	}
}

func TestCompatibilityDoesNotInventRequiredCategories(t *testing.T) {
	snapshot := testSnapshot()
	for _, components := range [][]Component{nil, snapshot.Products[:1], snapshot.Products[1:2]} {
		if issues := CheckCompatibility(components, nil); len(issues) != 0 {
			t.Fatalf("partial draft issues=%+v", issues)
		}
	}
}
