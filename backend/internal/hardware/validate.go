package hardware

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// These validate stored data shape/enums only, never compatibility or fitness.
func enum(value *string, allowed ...string) error {
	if value == nil {
		return nil
	}
	for _, choice := range allowed {
		if *value == choice {
			return nil
		}
	}
	return fmt.Errorf("invalid metadata enum %q", *value)
}

func positive[T ~int64 | ~float64](value *T) error {
	if value != nil && *value <= 0 {
		return fmt.Errorf("metadata value must be positive")
	}
	return nil
}

func nonnegative(value *float64) error {
	if value != nil && *value < 0 {
		return fmt.Errorf("metadata value must be nonnegative")
	}
	return nil
}

func links(modes []LinkMode) error {
	for _, m := range modes {
		if m.PCIEGeneration < 1 {
			return fmt.Errorf("invalid PCIe generation")
		}
		switch m.Lanes {
		case 1, 2, 4, 8, 16:
		default:
			return fmt.Errorf("invalid PCIe lane count")
		}
	}
	return nil
}

func connector(value *string) error {
	return enum(value, "pcie_6pin", "pcie_8pin", "12vhpwr", "12v_2x6", "sata_power", "none", "other")
}

func connection(value *string) error {
	return enum(value, "sata", "pcie_nvme", "pcie_other", "sas", "usb", "other")
}

func first(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

func provenance(p Provenance, allowSynthetic bool) error {
	ids := map[string]bool{}
	for _, source := range p.Sources {
		u, err := url.Parse(source.URL)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
			return fmt.Errorf("invalid source URL")
		}
		if !allowSynthetic && (strings.Contains(strings.ToLower(source.SourceID), "synthetic") || strings.Contains(strings.ToLower(source.Manufacturer), "synthetic") || strings.HasSuffix(strings.ToLower(u.Hostname()), ".invalid")) {
			return fmt.Errorf("synthetic evidence is local-only")
		}
		if source.SourceID == "" || ids[source.SourceID] || strings.TrimSpace(source.Manufacturer) == "" || strings.TrimSpace(source.Title) == "" || strings.TrimSpace(source.Section) == "" {
			return fmt.Errorf("invalid or duplicate metadata source")
		}
		if _, err := time.Parse(time.RFC3339, source.CheckedAt); err != nil {
			return fmt.Errorf("invalid source checked_at")
		}
		ids[source.SourceID] = true
	}
	for _, e := range p.Evidence {
		if len(e.Paths) == 0 || len(e.SourceIDs) == 0 {
			return fmt.Errorf("empty evidence reference")
		}
		for _, path := range e.Paths {
			if !strings.HasPrefix(path, "/") {
				return fmt.Errorf("evidence path must be a JSON Pointer")
			}
		}
		for _, id := range e.SourceIDs {
			if !ids[id] {
				return fmt.Errorf("evidence references missing source")
			}
		}
	}
	return nil
}

func ValidateGPU(g *GPU, allowSynthetic bool) error {
	if g == nil {
		return nil
	}
	if err := provenance(g.Provenance, allowSynthetic); err != nil {
		return err
	}
	if g.Display != nil {
		if err := positive(g.Display.VRAMCapacityBytes); err != nil {
			return err
		}
	}
	if g.Host != nil {
		if err := first(enum(g.Host.Connector, "pcie_x16", "other"), links(g.Host.SupportedLinkModes)); err != nil {
			return err
		}
	}
	if g.Power != nil {
		if err := first(positive(g.Power.BoardPowerW), positive(g.Power.MinSystemPowerW)); err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, option := range g.Power.ConnectorOptions {
			if option.OptionID == "" || seen[option.OptionID] || len(option.Connectors) == 0 {
				return fmt.Errorf("invalid power option")
			}
			seen[option.OptionID] = true
			if err := enum(&option.CablePolicy, "independent_cables", "manufacturer_cable_kit", "unspecified"); err != nil {
				return err
			}
			for _, c := range option.Connectors {
				if err := connector(&c.Type); err != nil {
					return err
				}
				if (c.Type == "none" && c.Count != 0) || (c.Type != "none" && c.Count < 1) {
					return fmt.Errorf("invalid power connector count")
				}
			}
		}
	}
	if d := g.Dimensions; d != nil {
		return first(positive(d.LengthMM), positive(d.HeightMM), positive(d.ThicknessMM), positive(d.OccupiedSlots))
	}
	return nil
}

func ValidateStorage(s *Storage, allowSynthetic bool) error {
	if s == nil {
		return nil
	}
	if err := provenance(s.Provenance, allowSynthetic); err != nil {
		return err
	}
	if d := s.Display; d != nil {
		if err := first(enum(d.MediaType, "ssd", "hdd", "other"), positive(d.CapacityBytes), nonnegative(d.SequentialReadMBS), nonnegative(d.SequentialWriteMBS)); err != nil {
			return err
		}
	}
	if h := s.Host; h != nil {
		if err := first(connection(h.ConnectionType), enum(h.Connector, "sata_data", "m2", "pcie_edge", "u2", "usb", "other"),
			enum(h.FormFactor, "m2", "2_5_inch", "3_5_inch", "pcie_aic", "u2", "other"), enum(h.M2Keying, "b", "m", "b_m"),
			positive(h.WidthMM), positive(h.LengthMM), links(h.SupportedLinkModes)); err != nil {
			return err
		}
	}
	if p := s.Power; p != nil {
		if err := first(connector(p.ConnectorType), nonnegative(p.MaxOperatingPowerW), nonnegative(p.StartupPowerW)); err != nil {
			return err
		}
	}
	if d := s.Dimensions; d != nil {
		return first(positive(d.WidthMM), positive(d.LengthMM), positive(d.HeightMM))
	}
	return nil
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func ValidateMotherboard(m *Motherboard, allowSynthetic bool) error {
	if m == nil {
		return nil
	}
	if err := provenance(m.Provenance, allowSynthetic); err != nil {
		return err
	}
	sources := map[string]bool{}
	for _, s := range m.Sources {
		sources[s.SourceID] = true
	}
	profiles := map[string]bool{}
	for _, p := range m.ConnectivityProfiles {
		if p.ProfileID == "" || profiles[p.ProfileID] {
			return fmt.Errorf("invalid profile ID")
		}
		profiles[p.ProfileID] = true
		if p.CPUProductIDs != nil && len(p.CPUProductIDs) == 0 {
			return fmt.Errorf("empty CPU profile is not a wildcard")
		}
		for _, id := range p.CPUProductIDs {
			if !uuidPattern.MatchString(id) {
				return fmt.Errorf("invalid profile CPU UUID")
			}
		}
		endpoints := map[string]bool{}
		for _, slot := range p.PCIESlots {
			if slot.SlotID == "" || endpoints[slot.SlotID] {
				return fmt.Errorf("invalid slot ID")
			}
			endpoints[slot.SlotID] = true
			if err := first(enum(slot.Connector, "pcie_x1", "pcie_x4", "pcie_x8", "pcie_x16"), links(slot.SupportedLinkModes)); err != nil {
				return err
			}
		}
		for _, port := range p.StoragePorts {
			if port.PortID == "" || endpoints[port.PortID] {
				return fmt.Errorf("invalid port ID")
			}
			endpoints[port.PortID] = true
			if err := first(enum(port.Connector, "sata_data", "m2", "other"), links(port.SupportedLinkModes)); err != nil {
				return err
			}
			for _, kind := range port.SupportedConnectionTypes {
				if err := connection(&kind); err != nil {
					return err
				}
			}
			for _, key := range port.AcceptedModuleKeys {
				if err := enum(&key, "b", "m", "b_m"); err != nil {
					return err
				}
			}
			for _, width := range port.SupportedWidthsMM {
				if err := positive(&width); err != nil {
					return err
				}
			}
			for _, length := range port.SupportedLengthsMM {
				if err := positive(&length); err != nil {
					return err
				}
			}
		}
		rules := map[string]bool{}
		for _, rule := range p.ResourceRules {
			if rule.RuleID == "" || rules[rule.RuleID] || len(rule.WhenAllOccupied) == 0 || len(rule.AffectedEndpointIDs) == 0 || len(rule.SourceIDs) == 0 {
				return fmt.Errorf("invalid resource rule")
			}
			rules[rule.RuleID] = true
			if err := enum(&rule.Effect, "disable", "restrict_link_modes", "requires_external_context"); err != nil {
				return err
			}
			for _, id := range append(append([]string{}, rule.WhenAllOccupied...), rule.AffectedEndpointIDs...) {
				if !endpoints[id] {
					return fmt.Errorf("rule references missing endpoint")
				}
			}
			for _, id := range rule.SourceIDs {
				if !sources[id] {
					return fmt.Errorf("rule references missing source")
				}
			}
			if rule.Effect == "restrict_link_modes" {
				if len(rule.AllowedLinkModes) == 0 {
					return fmt.Errorf("missing restricted modes")
				}
				if err := links(rule.AllowedLinkModes); err != nil {
					return err
				}
			} else if rule.AllowedLinkModes != nil {
				return fmt.Errorf("link modes not applicable to rule")
			}
			if rule.Effect == "requires_external_context" {
				if len(rule.RequiredContext) == 0 {
					return fmt.Errorf("missing required context")
				}
			} else if rule.RequiredContext == nil || len(rule.RequiredContext) != 0 {
				return fmt.Errorf("non-context rule requires empty context array")
			}
			for _, key := range rule.RequiredContext {
				if err := enum(&key, "bios_version", "usb4_usage", "unlisted_slot_occupancy"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
