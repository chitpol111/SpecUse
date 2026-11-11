package build

import (
	"encoding/json"
	"sort"
	"strings"

	"specuse/backend/internal/hardware"
)

const GPUStorageMode = "gpu_storage_v1"

type FieldReference struct {
	Location  string  `json:"location"`
	Path      string  `json:"path"`
	ProductID *string `json:"product_id"`
	Category  *string `json:"category"`
}

type Check struct {
	Code              string           `json:"code"`
	Scope             string           `json:"scope"`
	Status            string           `json:"status"`
	ReasonCode        string           `json:"reason_code"`
	Message           string           `json:"message"`
	ProductIDs        []string         `json:"product_ids"`
	MissingFields     []FieldReference `json:"missing_fields"`
	UnsupportedFields []FieldReference `json:"unsupported_fields"`
}

type Notice struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Compatibility struct {
	Mode              string   `json:"mode"`
	Status            string   `json:"status"`
	Checks            []Check  `json:"checks"`
	Warnings          []Notice `json:"warnings"`
	UnverifiedDomains []string `json:"unverified_domains"`
	BuildCompleteness string   `json:"build_completeness"`
}

func reference(c Component, path string) FieldReference {
	id, category := c.Product.ID, c.Product.Category
	return FieldReference{Location: "catalog", Path: path, ProductID: &id, Category: &category}
}

func missingComponent(category string) FieldReference {
	return FieldReference{Location: "request", Path: "/items", Category: &category}
}

func uniqueRefs(refs []FieldReference) []FieldReference {
	key := func(r FieldReference) string {
		id, category := "", ""
		if r.ProductID != nil {
			id = *r.ProductID
		}
		if r.Category != nil {
			category = *r.Category
		}
		return r.Location + "\x00" + id + "\x00" + category + "\x00" + r.Path
	}
	out := []FieldReference{}
	seen := map[string]bool{}
	for _, r := range refs {
		if !seen[key(r)] {
			out = append(out, r)
			seen[key(r)] = true
		}
	}
	sort.Slice(out, func(i, j int) bool { return key(out[i]) < key(out[j]) })
	return out
}

func productIDs(components []Component) []string {
	ids := []string{}
	seen := map[string]bool{}
	for _, c := range components {
		if !seen[c.Product.ID] {
			ids = append(ids, c.Product.ID)
			seen[c.Product.ID] = true
		}
	}
	sort.Strings(ids)
	return ids
}

// missingMetadata reports null/absent values, not unimplemented algorithms.
// It stops at the first missing parent and does not mark inapplicable fields.
func missingMetadata(c Component, value any, paths ...string) []FieldReference {
	encoded, _ := json.Marshal(value)
	var root any
	_ = json.Unmarshal(encoded, &root)
	refs := []FieldReference{}
	for _, path := range paths {
		node := root
		actual := "/specs"
		for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
			if node == nil {
				break
			}
			object, ok := node.(map[string]any)
			if !ok {
				break
			}
			actual += "/" + part
			node = object[part]
		}
		if node == nil {
			refs = append(refs, reference(c, actual))
		}
	}
	return uniqueRefs(refs)
}

func metadataRefs(c Component, domain string) []FieldReference {
	base := []string{"/sources", "/evidence"}
	switch c.Product.Category {
	case "gpu":
		g := c.Specs.GPU
		if g == nil {
			g = &hardware.GPU{}
		}
		paths := []string{"/host/connector", "/host/supported_link_modes"}
		if domain == "psu" {
			paths = []string{"/power/board_power_w", "/power/min_system_power_w", "/power/connector_options"}
		}
		if domain == "case" {
			paths = []string{"/dimensions/length_mm", "/dimensions/height_mm", "/dimensions/thickness_mm", "/dimensions/occupied_slots"}
		}
		return missingMetadata(c, g, append(base, paths...)...)
	case "storage":
		s := c.Specs.Storage
		if s == nil {
			s = &hardware.Storage{}
		}
		paths := []string{"/host/connection_type", "/host/connector", "/host/form_factor"}
		if s.Host != nil && s.Host.Connector != nil && *s.Host.Connector == "m2" {
			paths = append(paths, "/host/m2_keying", "/host/width_mm", "/host/length_mm")
		}
		if s.Host != nil && s.Host.ConnectionType != nil && *s.Host.ConnectionType == "pcie_nvme" {
			paths = append(paths, "/host/supported_link_modes")
		}
		if domain == "psu" {
			paths = []string{"/power/connector_type", "/power/max_operating_power_w", "/power/startup_power_w"}
		}
		if domain == "case" {
			paths = []string{"/host/form_factor", "/dimensions/width_mm", "/dimensions/length_mm", "/dimensions/height_mm"}
		}
		return missingMetadata(c, s, append(base, paths...)...)
	case "motherboard":
		if domain != "motherboard" {
			return []FieldReference{}
		}
		m := c.Specs.Motherboard
		if m == nil {
			m = &hardware.Motherboard{}
		}
		return missingMetadata(c, m, append(base, "/connectivity_profiles")...)
	}
	return []FieldReference{}
}

// Unsupported class reporting is coverage, not a hardware compatibility check.
func unsupportedDevice(c Component) []FieldReference {
	refs := []FieldReference{}
	if c.Product.Category == "gpu" {
		if g := c.Specs.GPU; g != nil && g.Host != nil && g.Host.Connector != nil && *g.Host.Connector == "other" {
			refs = append(refs, reference(c, "/specs/host/connector"))
		}
		return refs
	}
	s := c.Specs.Storage
	if s == nil || s.Host == nil {
		return refs
	}
	h := s.Host
	if h.ConnectionType != nil && *h.ConnectionType != "sata" && *h.ConnectionType != "pcie_nvme" {
		refs = append(refs, reference(c, "/specs/host/connection_type"))
	}
	if h.Connector != nil && *h.Connector != "sata_data" && *h.Connector != "m2" {
		refs = append(refs, reference(c, "/specs/host/connector"))
	}
	if h.FormFactor != nil && *h.FormFactor != "m2" && *h.FormFactor != "2_5_inch" && *h.FormFactor != "3_5_inch" {
		refs = append(refs, reference(c, "/specs/host/form_factor"))
	}
	return uniqueRefs(refs)
}

func newCheck(code, scope string, components []Component) Check {
	return Check{Code: code, Scope: scope, Status: "unknown", ReasonCode: "NOT_IMPLEMENTED",
		Message: "ยังไม่เปิดการตรวจฮาร์ดแวร์ส่วนนี้ในช่วง A", ProductIDs: productIDs(components),
		MissingFields: []FieldReference{}, UnsupportedFields: []FieldReference{}}
}

func aggregate(checks []Check) string {
	unknown := len(checks) == 0
	for _, c := range checks {
		if c.Status == "failed" {
			return "failed"
		}
		if c.Status != "passed" {
			unknown = true
		}
	}
	if unknown {
		return "unknown"
	}
	return "passed"
}

func phaseACompatibility(selected []Component, models []RAMModel, ramSKUs []Component) *Compatibility {
	groups := map[string][]Component{}
	for _, c := range selected {
		groups[c.Product.Category] = append(groups[c.Product.Category], c)
	}
	all := append(append([]Component{}, selected...), ramSKUs...)
	result := &Compatibility{Mode: GPUStorageMode, Checks: []Check{}, Warnings: []Notice{}, UnverifiedDomains: []string{}, BuildCompleteness: "not_evaluated"}
	boards, cpus := groups["motherboard"], groups["cpu"]
	if len(cpus) > 0 && len(boards) > 0 {
		c := newCheck("CPU_SOCKET", "basic", append(append([]Component{}, cpus...), boards...))
		c.Status, c.ReasonCode, c.Message = "passed", "MATCH", "CPU socket ตรงกันตาม catalog; ไม่ใช่การตรวจ CPU support/BIOS ทั้งหมด"
		for _, part := range append(append([]Component{}, cpus...), boards...) {
			if part.Specs.CPUSocket == "" {
				c.MissingFields = append(c.MissingFields, reference(part, "/specs/cpu_socket"))
			}
		}
		for _, cpu := range cpus {
			for _, board := range boards {
				if cpu.Specs.CPUSocket != "" && board.Specs.CPUSocket != "" && cpu.Specs.CPUSocket != board.Specs.CPUSocket {
					c.Status, c.ReasonCode, c.Message = "failed", "INCOMPATIBLE", "CPU socket ไม่ตรงกับ motherboard"
				}
			}
		}
		if c.Status != "failed" && len(c.MissingFields) > 0 {
			c.Status, c.ReasonCode, c.Message = "unknown", "METADATA_MISSING", "ข้อมูล CPU socket ไม่ครบ"
		}
		c.MissingFields = uniqueRefs(c.MissingFields)
		result.Checks = append(result.Checks, c)
	}
	if len(models) > 0 {
		c := newCheck("RAM_TYPE", "basic", append(append([]Component{}, ramSKUs...), boards...))
		c.Status, c.ReasonCode, c.Message = "passed", "MATCH", "RAM type ตรงกับ motherboard ตาม catalog"
		if len(boards) == 0 {
			c.MissingFields = append(c.MissingFields, missingComponent("motherboard"))
		}
		for _, board := range boards {
			if board.Specs.RAMType == "" {
				c.MissingFields = append(c.MissingFields, reference(board, "/specs/ram_type"))
			}
		}
		for _, model := range models {
			if model.RAMType == "" {
				category := "ram"
				c.MissingFields = append(c.MissingFields, FieldReference{Location: "context", Path: "/ram_models/" + model.ID + "/ram_type", Category: &category})
			}
			for _, board := range boards {
				if model.RAMType != "" && board.Specs.RAMType != "" && model.RAMType != board.Specs.RAMType {
					c.Status, c.ReasonCode, c.Message = "failed", "INCOMPATIBLE", "RAM type ไม่ตรงกับ motherboard"
				}
			}
		}
		if c.Status != "failed" && len(c.MissingFields) > 0 {
			c.Status, c.ReasonCode, c.Message = "unknown", "METADATA_MISSING", "ยังไม่มี motherboard หรือข้อมูล RAM type ที่จำเป็น"
			if len(boards) == 0 {
				c.ReasonCode = "MISSING_COMPONENT"
			}
		}
		c.MissingFields = uniqueRefs(c.MissingFields)
		result.Checks = append(result.Checks, c)
	}
	appendPending := func(code, scope string, components []Component, needsBoard bool) {
		c := newCheck(code, scope, components)
		for _, part := range components {
			c.MissingFields = append(c.MissingFields, metadataRefs(part, scope)...)
			if scope == "motherboard" {
				c.UnsupportedFields = append(c.UnsupportedFields, unsupportedDevice(part)...)
			}
		}
		if needsBoard {
			if len(boards) == 0 {
				c.MissingFields = append(c.MissingFields, missingComponent("motherboard"))
			}
			if len(cpus) == 0 {
				c.MissingFields = append(c.MissingFields, missingComponent("cpu"))
			}
		}
		if scope == "psu" || scope == "case" {
			if len(groups[scope]) == 0 {
				c.MissingFields = append(c.MissingFields, missingComponent(scope))
			}
		}
		if code == "SYSTEM_PSU" {
			c.MissingFields = append(c.MissingFields, FieldReference{Location: "context", Path: "/system_power_envelope"})
		}
		if code == "SYSTEM_CASE" {
			c.MissingFields = append(c.MissingFields, FieldReference{Location: "context", Path: "/installation_context"})
		}
		c.MissingFields = uniqueRefs(c.MissingFields)
		c.UnsupportedFields = uniqueRefs(c.UnsupportedFields)
		if len(c.UnsupportedFields) > 0 {
			c.ReasonCode = "UNSUPPORTED_DEVICE_CLASS"
			c.Message = "ทราบชนิดอุปกรณ์ แต่ยังไม่รองรับการตรวจ class นี้"
		}
		result.Checks = append(result.Checks, c)
	}
	if len(groups["gpu"]) > 0 {
		appendPending("GPU_MOTHERBOARD", "motherboard", append(append([]Component{}, groups["gpu"]...), boards...), true)
	}
	if len(groups["storage"]) > 0 {
		appendPending("STORAGE_MOTHERBOARD", "motherboard", append(append([]Component{}, groups["storage"]...), boards...), true)
	}
	if len(groups["gpu"])+len(groups["storage"]) > 0 {
		parts := append(append(append([]Component{}, groups["gpu"]...), groups["storage"]...), boards...)
		appendPending("MOTHERBOARD_RESOURCES", "motherboard", parts, true)
	}
	for _, category := range []string{"gpu", "storage"} {
		if len(groups[category]) > 0 {
			for _, scope := range []string{"psu", "case"} {
				parts := append(append([]Component{}, groups[category]...), groups[scope]...)
				appendPending(strings.ToUpper(category)+"_"+strings.ToUpper(scope), scope, parts, false)
			}
		}
	}
	if len(all) > 0 {
		appendPending("SYSTEM_PSU", "psu", all, false)
		appendPending("SYSTEM_CASE", "case", all, false)
	}
	unknownDomains := map[string]bool{}
	for _, c := range result.Checks {
		if c.Status == "unknown" {
			unknownDomains[c.Scope] = true
		}
	}
	for scope := range unknownDomains {
		result.UnverifiedDomains = append(result.UnverifiedDomains, scope)
	}
	sort.Strings(result.UnverifiedDomains)
	result.Status = aggregate(result.Checks)
	return result
}
