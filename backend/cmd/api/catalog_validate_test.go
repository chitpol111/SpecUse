package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"specuse/backend/internal/catalog"
)

// The frontend can build requests entirely from GET data without maintaining
// product/model/socket/type mappings or receiving the fixture's private data.
func TestCatalogMetadataRoundTripsToBuildValidation(t *testing.T) {
	h := fixtureAPI(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/products", nil))
	var response struct {
		Data catalog.Page `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 {
		t.Fatalf("catalog status=%d", w.Code)
	}
	byID := map[string]catalog.ProductDetails{}
	var board, matchingCPU, otherCPU, single, pair, otherRAM catalog.ProductDetails
	for _, product := range response.Data.Items {
		byID[product.ID] = product
		if product.Category == "motherboard" {
			board = product
		}
	}
	if board.Specs == nil {
		t.Fatal("motherboard specs missing")
	}
	for _, product := range response.Data.Items {
		if product.Specs == nil {
			t.Fatalf("missing specs for %s", product.ID)
		}
		switch product.Category {
		case "cpu":
			if product.Specs.CPUSocket == board.Specs.CPUSocket {
				matchingCPU = product
			} else {
				otherCPU = product
			}
		case "ram":
			if product.Specs.RAMType != board.Specs.RAMType {
				otherRAM = product
			}
			if product.Specs.RAMType == board.Specs.RAMType && product.Specs.ModulesPerPackage == 1 {
				single = product
			}
		}
	}
	for _, product := range response.Data.Items {
		if product.Category == "ram" && product.RAMModelID == single.RAMModelID && product.Specs.ModulesPerPackage == 2 {
			pair = product
		}
	}
	for _, product := range []catalog.ProductDetails{matchingCPU, otherCPU, single, pair, otherRAM} {
		if product.ID == "" {
			t.Fatal("fixture must exercise matching/mismatching CPU and RAM, including 1/2-module SKUs")
		}
	}
	for _, tt := range []struct {
		name       string
		cpu, ram   catalog.ProductDetails
		compatible bool
	}{
		{"compatible", matchingCPU, single, true},
		{"CPU socket mismatch", otherCPU, single, false},
		{"RAM type mismatch", matchingCPU, otherRAM, false},
		{"both mismatch", otherCPU, otherRAM, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]any{
				"items": []map[string]any{
					{"product_id": tt.cpu.ID, "quantity": 1, "owned_quantity": 1},
					{"product_id": board.ID, "quantity": 1, "owned_quantity": 0},
				},
				"ram_items": []map[string]any{{"ram_model_id": tt.ram.RAMModelID, "required_modules": 3, "owned_modules": 0}},
			}
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("POST", "/api/v1/builds/validate", strings.NewReader(string(body)))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			var result struct {
				Data struct {
					Compatible bool `json:"compatible"`
					Issues     []struct {
						Code string `json:"code"`
					} `json:"issues"`
					Items []struct {
						ProductID        string `json:"product_id"`
						UnitPriceTHB     int64  `json:"unit_price_thb"`
						PurchasePriceTHB int64  `json:"purchase_price_thb"`
					} `json:"items"`
					RAMSelections []struct {
						RAMModelID    string `json:"ram_model_id"`
						TotalPriceTHB int64  `json:"total_price_thb"`
						Packages      []struct {
							ProductID         string `json:"product_id"`
							Quantity          int64  `json:"quantity"`
							ModulesPerPackage int64  `json:"modules_per_package"`
							UnitPriceTHB      int64  `json:"unit_price_thb"`
						} `json:"packages"`
					} `json:"ram_selections"`
					TotalPriceTHB int64 `json:"total_price_thb"`
				} `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			data := result.Data
			if w.Code != 200 || data.Compatible != tt.compatible || len(data.Items) != 2 || len(data.RAMSelections) != 1 {
				t.Fatalf("status=%d result=%s", w.Code, w.Body.String())
			}
			if data.Items[0].ProductID != tt.cpu.ID || data.Items[0].UnitPriceTHB != tt.cpu.PriceTHB || data.Items[0].PurchasePriceTHB != 0 || data.Items[1].PurchasePriceTHB != board.PriceTHB {
				t.Fatalf("catalog prices/owned quantities drifted: %+v", data.Items)
			}
			foundCPU, foundRAM := false, false
			for _, issue := range data.Issues {
				if issue.Code == "CPU_SOCKET_MISMATCH" {
					foundCPU = true
				}
				if issue.Code == "RAM_TYPE_MISMATCH" {
					foundRAM = true
				}
			}
			if foundCPU != (tt.cpu.Specs.CPUSocket != board.Specs.CPUSocket) || foundRAM != (tt.ram.Specs.RAMType != board.Specs.RAMType) {
				t.Fatalf("GET specs and POST issues disagree: %+v", data.Issues)
			}
			selection := data.RAMSelections[0]
			if selection.RAMModelID != tt.ram.RAMModelID {
				t.Fatal("model ID changed between catalog and validation")
			}
			var ramTotal int64
			for _, pack := range selection.Packages {
				product, exists := byID[pack.ProductID]
				if !exists || product.RAMModelID != selection.RAMModelID || product.Specs.RAMType != tt.ram.Specs.RAMType || product.Specs.ModulesPerPackage != pack.ModulesPerPackage || product.PriceTHB != pack.UnitPriceTHB {
					t.Fatalf("catalog SKU/specs disagree with selection: %+v", pack)
				}
				ramTotal += pack.Quantity * product.PriceTHB
			}
			if selection.TotalPriceTHB != ramTotal || data.TotalPriceTHB != board.PriceTHB+ramTotal {
				t.Fatalf("catalog prices and validation total disagree: %s", w.Body.String())
			}
			if tt.compatible && (len(selection.Packages) != 2 || data.TotalPriceTHB != 11500) {
				t.Fatalf("existing success example changed: %s", w.Body.String())
			}
		})
	}
}
