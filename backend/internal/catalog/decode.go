package catalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"specuse/backend/internal/hardware"
)

func decodeStrict(data []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("expected one JSON value")
	}
	return nil
}

// ProductDetails knows the category before decoding its typed specs. This is
// also used by the local fixture loader; unknown fixture fields fail startup.
func (p *ProductDetails) UnmarshalJSON(data []byte) error {
	var raw struct {
		Product
		Specs      json.RawMessage `json:"specs"`
		RAMModelID string          `json:"ram_model_id"`
	}
	if err := decodeStrict(data, &raw); err != nil {
		return err
	}
	*p = ProductDetails{Product: raw.Product, RAMModelID: raw.RAMModelID}
	if len(raw.Specs) == 0 || bytes.Equal(bytes.TrimSpace(raw.Specs), []byte("null")) {
		return nil
	}
	s := &Specs{}
	switch raw.Category {
	case "gpu":
		s.GPU = &hardware.GPU{}
		if err := decodeStrict(raw.Specs, s.GPU); err != nil {
			return err
		}
	case "storage":
		s.Storage = &hardware.Storage{}
		if err := decodeStrict(raw.Specs, s.Storage); err != nil {
			return err
		}
	case "motherboard":
		var board struct {
			CPUSocket string `json:"cpu_socket"`
			RAMType   string `json:"ram_type"`
			hardware.Motherboard
		}
		if err := decodeStrict(raw.Specs, &board); err != nil {
			return err
		}
		s.CPUSocket, s.RAMType, s.Motherboard = board.CPUSocket, board.RAMType, &board.Motherboard
	default:
		var legacy struct {
			CPUSocket         string `json:"cpu_socket"`
			RAMType           string `json:"ram_type"`
			ModulesPerPackage int64  `json:"modules_per_package"`
		}
		if err := decodeStrict(raw.Specs, &legacy); err != nil {
			return err
		}
		s.CPUSocket, s.RAMType, s.ModulesPerPackage = legacy.CPUSocket, legacy.RAMType, legacy.ModulesPerPackage
	}
	p.Specs = s
	return nil
}
