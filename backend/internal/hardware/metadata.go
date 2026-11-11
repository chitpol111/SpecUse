// Package hardware holds nullable catalog metadata, not hardware-check logic.
// A nil pointer/slice means unknown; a non-nil empty slice means reviewed empty.
package hardware

import "encoding/json"

type LinkMode struct {
	PCIEGeneration int64 `json:"pcie_generation"`
	Lanes          int64 `json:"lanes"`
}

type Source struct {
	SourceID         string  `json:"source_id"`
	Manufacturer     string  `json:"manufacturer"`
	Title            string  `json:"title"`
	URL              string  `json:"url"`
	DocumentRevision *string `json:"document_revision"`
	Section          string  `json:"section"`
	CheckedAt        string  `json:"checked_at"`
}

type Evidence struct {
	Paths     []string `json:"paths"`
	SourceIDs []string `json:"source_ids"`
}

type Provenance struct {
	Sources  []Source   `json:"sources"`
	Evidence []Evidence `json:"evidence"`
}

type GPU struct {
	Display    *GPUDisplay    `json:"display"`
	Host       *GPUHost       `json:"host"`
	Power      *GPUPower      `json:"power"`
	Dimensions *GPUDimensions `json:"dimensions"`
	Provenance
}

type GPUDisplay struct {
	ChipsetName       *string `json:"chipset_name"`
	VRAMCapacityBytes *int64  `json:"vram_capacity_bytes"`
}

type GPUHost struct {
	Connector          *string    `json:"connector"`
	SupportedLinkModes []LinkMode `json:"supported_link_modes"`
}

type GPUPower struct {
	BoardPowerW      *float64      `json:"board_power_w"`
	MinSystemPowerW  *int64        `json:"min_system_power_w"`
	ConnectorOptions []PowerOption `json:"connector_options"`
}

type PowerOption struct {
	OptionID          string           `json:"option_id"`
	AdapterPartNumber *string          `json:"adapter_part_number"`
	Connectors        []PowerConnector `json:"connectors"`
	CablePolicy       string           `json:"cable_policy"`
}

type PowerConnector struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

type GPUDimensions struct {
	LengthMM      *float64 `json:"length_mm"`
	HeightMM      *float64 `json:"height_mm"`
	ThicknessMM   *float64 `json:"thickness_mm"`
	OccupiedSlots *int64   `json:"occupied_slots"`
}

type Storage struct {
	Display    *StorageDisplay    `json:"display"`
	Host       *StorageHost       `json:"host"`
	Power      *StoragePower      `json:"power"`
	Dimensions *StorageDimensions `json:"dimensions"`
	Provenance
}

type StorageDisplay struct {
	MediaType          *string  `json:"media_type"`
	CapacityBytes      *int64   `json:"capacity_bytes"`
	SequentialReadMBS  *float64 `json:"sequential_read_mb_s"`
	SequentialWriteMBS *float64 `json:"sequential_write_mb_s"`
}

type StorageHost struct {
	ConnectionType     *string    `json:"connection_type"`
	Connector          *string    `json:"connector"`
	FormFactor         *string    `json:"form_factor"`
	M2Keying           *string    `json:"m2_keying"`
	WidthMM            *int64     `json:"width_mm"`
	LengthMM           *int64     `json:"length_mm"`
	SupportedLinkModes []LinkMode `json:"supported_link_modes"`
}

type StoragePower struct {
	ConnectorType      *string  `json:"connector_type"`
	MaxOperatingPowerW *float64 `json:"max_operating_power_w"`
	StartupPowerW      *float64 `json:"startup_power_w"`
}

type StorageDimensions struct {
	WidthMM  *float64 `json:"width_mm"`
	LengthMM *float64 `json:"length_mm"`
	HeightMM *float64 `json:"height_mm"`
}

// Motherboard adds connectivity/provenance to the existing socket/RAM fields.
type Motherboard struct {
	ConnectivityProfiles []IOProfile `json:"connectivity_profiles"`
	Provenance
}

type IOProfile struct {
	ProfileID                    string         `json:"profile_id"`
	CPUProductIDs                []string       `json:"cpu_product_ids"`
	RequiresFirmwareConfirmation *bool          `json:"requires_firmware_confirmation"`
	PCIESlots                    []PCIESlot     `json:"pcie_slots"`
	StoragePorts                 []StoragePort  `json:"storage_ports"`
	ResourceRules                []ResourceRule `json:"resource_rules"`
}

type PCIESlot struct {
	SlotID             string     `json:"slot_id"`
	Connector          *string    `json:"connector"`
	SupportedLinkModes []LinkMode `json:"supported_link_modes"`
	GPUSupported       *bool      `json:"gpu_supported"`
}

type StoragePort struct {
	PortID                   string     `json:"port_id"`
	Connector                *string    `json:"connector"`
	SupportedConnectionTypes []string   `json:"supported_connection_types"`
	AcceptedModuleKeys       []string   `json:"accepted_module_keys"`
	SupportedWidthsMM        []int64    `json:"supported_widths_mm"`
	SupportedLengthsMM       []int64    `json:"supported_lengths_mm"`
	SupportedLinkModes       []LinkMode `json:"supported_link_modes"`
}

type ResourceRule struct {
	RuleID              string     `json:"rule_id"`
	WhenAllOccupied     []string   `json:"when_all_occupied"`
	Effect              string     `json:"effect"`
	AffectedEndpointIDs []string   `json:"affected_endpoint_ids"`
	AllowedLinkModes    []LinkMode `json:"allowed_link_modes"`
	RequiredContext     []string   `json:"required_context"`
	SourceIDs           []string   `json:"source_ids"`
}

// Clone makes nested pointers/slices independent at repository boundaries.
// JSON round-tripping preserves int64 values and the distinction null vs [].
func Clone[T any](source T) (T, error) {
	var target T
	data, err := json.Marshal(source)
	if err != nil {
		return target, err
	}
	err = json.Unmarshal(data, &target)
	return target, err
}
