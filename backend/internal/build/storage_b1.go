package build

import (
	"fmt"
	"sort"
	"strings"

	"specuse/backend/internal/hardware"
)

// B1 proves direct internal SATA allocations only. It does not evaluate GPU,
// M.2, sharing rules, BIOS, PSU or case compatibility. Bindings stay internal.
type sataAssignment struct {
	ProductID string
	Instance  int64
	PortID    string
}

func evidenceCovers(p hardware.Provenance, path string) bool {
	sources := map[string]bool{}
	for _, s := range p.Sources {
		sources[s.SourceID] = true
	}
	for _, e := range p.Evidence {
		valid := len(e.SourceIDs) > 0
		for _, id := range e.SourceIDs {
			if !sources[id] {
				valid = false
			}
		}
		if !valid {
			continue
		}
		for _, covered := range e.Paths {
			if path == covered || strings.HasPrefix(path, covered+"/") {
				return true
			}
		}
	}
	return false
}

// Require coverage of the actual claims, including complete lists before
// using their length as proof. A non-null evidence array alone is insufficient.
func requireEvidence(c Component, p hardware.Provenance, paths ...string) []FieldReference {
	refs := []FieldReference{}
	if len(p.Sources) == 0 {
		refs = append(refs, reference(c, "/specs/sources"))
	}
	for _, path := range paths {
		if !evidenceCovers(p, path) {
			refs = append(refs, reference(c, "/specs/evidence"))
			break
		}
	}
	return uniqueRefs(refs)
}

func allocateSATA(items []PricedItem, ports []string) ([]sataAssignment, bool) {
	rows := append([]PricedItem{}, items...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].ProductID < rows[j].ProductID })
	orderedPorts := append([]string{}, ports...)
	sort.Strings(orderedPorts)
	seen := map[string]bool{}
	for _, id := range orderedPorts {
		if id == "" || seen[id] {
			return nil, false
		}
		seen[id] = true
	}
	remaining := int64(len(orderedPorts))
	for _, row := range rows {
		if row.Quantity < 1 || row.Quantity > remaining {
			return nil, false
		}
		remaining -= row.Quantity // bounded; never expand MaxInt64 quantities
	}
	out := []sataAssignment{}
	for _, row := range rows {
		for n := int64(1); n <= row.Quantity; n++ {
			out = append(out, sataAssignment{row.ProductID, n, orderedPorts[len(out)]})
		}
	}
	return out, true
}

func storageB1Checks(selected []Component, items []PricedItem) (Check, Check, []Issue) {
	groups := map[string][]Component{}
	for _, c := range selected {
		groups[c.Product.Category] = append(groups[c.Product.Category], c)
	}
	parts := append(append(append([]Component{}, groups["storage"]...), groups["motherboard"]...), groups["cpu"]...)
	parts = append(parts, groups["gpu"]...)
	storage := newCheck("STORAGE_MOTHERBOARD", "motherboard", parts)
	resources := newCheck("MOTHERBOARD_RESOURCES", "motherboard", parts)
	unknown := func(reason, message string, missing, unsupported []FieldReference) (Check, Check, []Issue) {
		for _, c := range []*Check{&storage, &resources} {
			c.ReasonCode, c.Message = reason, message
			c.MissingFields = uniqueRefs(missing)
			c.UnsupportedFields = uniqueRefs(unsupported)
		}
		return storage, resources, []Issue{}
	}
	missing, unsupported := []FieldReference{}, []FieldReference{}
	deferred := len(groups["gpu"]) > 0
	for _, c := range groups["storage"] {
		unsupported = append(unsupported, unsupportedDevice(c)...)
		s := c.Specs.Storage
		if s == nil || s.Host == nil {
			missing = append(missing, reference(c, "/specs/host"))
			continue
		}
		h := s.Host
		if h.ConnectionType == nil {
			missing = append(missing, reference(c, "/specs/host/connection_type"))
		}
		if h.Connector == nil {
			missing = append(missing, reference(c, "/specs/host/connector"))
		}
		if h.FormFactor == nil {
			missing = append(missing, reference(c, "/specs/host/form_factor"))
		}
		if h.ConnectionType != nil && *h.ConnectionType != "sata" {
			deferred = true
		}
		if h.Connector != nil && *h.Connector != "sata_data" {
			deferred = true
		}
		if h.FormFactor != nil && *h.FormFactor != "2_5_inch" && *h.FormFactor != "3_5_inch" {
			deferred = true
		}
		missing = append(missing, requireEvidence(c, s.Provenance, "/host/connection_type", "/host/connector", "/host/form_factor")...)
	}
	if len(groups["cpu"]) == 0 {
		missing = append(missing, missingComponent("cpu"))
	}
	if len(groups["motherboard"]) == 0 {
		missing = append(missing, missingComponent("motherboard"))
	}
	if len(unsupported) > 0 {
		return unknown("UNSUPPORTED_DEVICE_CLASS", "มี Storage class ที่ยังไม่รองรับการตรวจ", missing, unsupported)
	}
	if deferred {
		return unknown("NOT_IMPLEMENTED", "B1 ยังไม่ประเมิน GPU/M.2 หรือ Storage ที่อยู่นอก direct SATA; ยังยืนยัน allocation ไม่ได้", missing, nil)
	}
	if len(groups["cpu"]) == 0 || len(groups["motherboard"]) == 0 {
		return unknown("MISSING_COMPONENT", "ต้องเลือก CPU และ motherboard เพื่อยืนยัน SATA profile", missing, nil)
	}
	counts := map[string]int64{}
	for _, row := range items {
		counts[row.ProductID] = row.Quantity
	}
	if len(groups["cpu"]) != 1 || len(groups["motherboard"]) != 1 || counts[groups["cpu"][0].Product.ID] != 1 || counts[groups["motherboard"][0].Product.ID] != 1 {
		return unknown("NOT_IMPLEMENTED", "B1 ประเมินเฉพาะ CPU และ motherboard อย่างละหนึ่ง; ไม่เพิ่มข้อจำกัดใหม่ให้ legacy", missing, nil)
	}
	cpu, board := groups["cpu"][0], groups["motherboard"][0]
	m := board.Specs.Motherboard
	if m == nil || m.ConnectivityProfiles == nil {
		missing = append(missing, reference(board, "/specs/connectivity_profiles"))
		return unknown("METADATA_MISSING", "ยังไม่มี motherboard connectivity profiles", missing, nil)
	}
	missing = append(missing, requireEvidence(board, m.Provenance, "/connectivity_profiles")...)
	match := -1
	ambiguous := false
	for i, p := range m.ConnectivityProfiles {
		if p.CPUProductIDs == nil {
			missing = append(missing, reference(board, fmt.Sprintf("/specs/connectivity_profiles/%d/cpu_product_ids", i)))
			ambiguous = true
		}
		for _, id := range p.CPUProductIDs {
			if strings.EqualFold(id, cpu.Product.ID) {
				if match != -1 {
					ambiguous = true
				}
				match = i
			}
		}
	}
	if match < 0 || ambiguous {
		return unknown("CONTEXT_MISSING", "ยังเลือก CPU-specific profile เดียวอย่างแน่นอนไม่ได้", missing, nil)
	}
	p := m.ConnectivityProfiles[match]
	prefix := fmt.Sprintf("/specs/connectivity_profiles/%d", match)
	if p.RequiresFirmwareConfirmation == nil {
		missing = append(missing, reference(board, prefix+"/requires_firmware_confirmation"))
		return unknown("METADATA_MISSING", "ยังไม่ทราบเงื่อนไข firmware ของ profile", missing, nil)
	}
	if *p.RequiresFirmwareConfirmation {
		id, category := board.Product.ID, "motherboard"
		missing = append(missing, FieldReference{Location: "context", Path: "/bios_version", ProductID: &id, Category: &category})
		return unknown("CONTEXT_MISSING", "profile ต้องยืนยัน BIOS/context ที่ B1 ยังไม่รองรับ", missing, nil)
	}
	if p.ResourceRules == nil {
		missing = append(missing, reference(board, prefix+"/resource_rules"))
	}
	if p.StoragePorts == nil {
		missing = append(missing, reference(board, prefix+"/storage_ports"))
	}
	if len(p.ResourceRules) > 0 {
		// B1 conservatively defers every nonempty rule set. It does not attempt
		// to prove a sharing rule irrelevant or apply its hardware effect.
		unsupported = append(unsupported, reference(board, prefix+"/resource_rules"))
		return unknown("NOT_IMPLEMENTED", "มี sharing/resource rules ที่ B1 ยังไม่ประเมิน; ห้ามนับพอร์ตดิบ", missing, unsupported)
	}
	ports := []string{}
	seenPorts := map[string]bool{}
	for i, port := range p.StoragePorts {
		path := fmt.Sprintf("%s/storage_ports/%d", prefix, i)
		if port.PortID == "" || seenPorts[port.PortID] {
			return unknown("METADATA_MISSING", "port IDs ไม่ครบหรือซ้ำ จึงยังพิสูจน์จำนวนพอร์ตไม่ได้", missing, nil)
		}
		seenPorts[port.PortID] = true
		if port.Connector == nil {
			missing = append(missing, reference(board, path+"/connector"))
			continue
		}
		if *port.Connector == "m2" {
			continue
		} // no adapter inference; not a direct SATA data port
		if *port.Connector != "sata_data" {
			unsupported = append(unsupported, reference(board, path+"/connector"))
			continue
		}
		if port.SupportedConnectionTypes == nil {
			missing = append(missing, reference(board, path+"/supported_connection_types"))
			continue
		}
		for _, kind := range port.SupportedConnectionTypes {
			if kind == "sata" {
				ports = append(ports, port.PortID)
				break
			}
		}
	}
	if len(unsupported) > 0 {
		return unknown("UNSUPPORTED_DEVICE_CLASS", "มี port class ที่อาจเปลี่ยน allocation แต่ยังตรวจไม่ได้", missing, unsupported)
	}
	if len(missing) > 0 {
		return unknown("METADATA_MISSING", "ข้อมูลหรือ evidence ที่จำเป็นต่อ SATA allocation ยังไม่ครบ", missing, nil)
	}
	rows := []PricedItem{}
	for _, c := range groups["storage"] {
		for _, row := range items {
			if row.ProductID == c.Product.ID {
				rows = append(rows, row)
			}
		}
	}
	assignments, ok := allocateSATA(rows, ports)
	if !ok {
		code := "MOTHERBOARD_RESOURCE_CONFLICT"
		message := "จำนวน SATA ที่ต้องติดตั้งรวม owned มากกว่าพอร์ตที่ยืนยันว่าใช้ได้"
		if len(ports) == 0 {
			code = "STORAGE_INTERFACE_MISMATCH"
			message = "profile ที่ตรวจครบไม่มีพอร์ต direct SATA ที่รองรับ connector/interface นี้"
		}
		for _, c := range []*Check{&storage, &resources} {
			c.Status, c.ReasonCode, c.Message = "failed", "INCOMPATIBLE", message
		}
		return storage, resources, []Issue{{Code: code, RightID: board.Product.ID}}
	}
	for _, c := range []*Check{&storage, &resources} {
		c.Status, c.ReasonCode, c.Message = "passed", "MATCH", fmt.Sprintf("จัด SATA %d ลูก รวม owned ลงพอร์ตต่างกันได้ตาม profile และ evidence ที่ตรวจครบ; ไม่รวม PSU/Case", len(assignments))
	}
	return storage, resources, []Issue{}
}

func applyStorageB1(result *Compatibility, selected []Component, items []PricedItem) []Issue {
	hasStorage := false
	for _, c := range selected {
		if c.Product.Category == "storage" {
			hasStorage = true
		}
	}
	if !hasStorage {
		return []Issue{}
	}
	storage, resources, issues := storageB1Checks(selected, items)
	for i, c := range result.Checks {
		switch c.Code {
		case "STORAGE_MOTHERBOARD":
			result.Checks[i] = storage
		case "MOTHERBOARD_RESOURCES":
			result.Checks[i] = resources
		}
	}
	return issues
}
