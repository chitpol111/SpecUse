package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/repository"
)

const validationBoardID = "abcdefab-0000-0000-0000-000000000004"
const validationSingleID = "abcdefab-0000-0000-0000-000000000005"
const validationPairID = "abcdefab-0000-0000-0000-000000000006"
const validationModelID = "10000000-0000-0000-0000-000000000001"
const validationDDR4ModelID = "10000000-0000-0000-0000-000000000002"

func validationSnapshot() build.Snapshot {
	return build.Snapshot{
		Products: []build.Component{
			{Product: catalog.Product{ID: activeID, Name: "CPU", Category: "cpu", Brand: "Example", PriceTHB: 25000, IsActive: true}, Specs: build.Specs{CPUSocket: "AM5"}},
			{Product: catalog.Product{ID: validationBoardID, Name: "Board", Category: "motherboard", Brand: "Example", PriceTHB: 5000, IsActive: true}, Specs: build.Specs{CPUSocket: "AM5", RAMType: "DDR5"}},
			{Product: catalog.Product{ID: validationSingleID, Name: "RAM single", Category: "ram", Brand: "Example", PriceTHB: 2500, IsActive: true}, Specs: build.Specs{RAMType: "DDR5", ModulesPerPackage: 1}, RAMModelID: validationModelID},
			{Product: catalog.Product{ID: validationPairID, Name: "RAM pair", Category: "ram", Brand: "Example", PriceTHB: 4000, IsActive: true}, Specs: build.Specs{RAMType: "DDR5", ModulesPerPackage: 2}, RAMModelID: validationModelID},
			{Product: catalog.Product{ID: "abcdefab-0000-0000-0000-000000000007", Name: "DDR4", Category: "ram", Brand: "Example", PriceTHB: 1500, IsActive: true}, Specs: build.Specs{RAMType: "DDR4", ModulesPerPackage: 1}, RAMModelID: validationDDR4ModelID},
		},
		RAMModels: []build.RAMModel{{ID: validationModelID, RAMType: "DDR5"}, {ID: validationDDR4ModelID, RAMType: "DDR4"}},
	}
}

func validationAPI(t *testing.T, snapshot build.Snapshot) http.Handler {
	t.Helper()
	repo, err := repository.NewBuildMemory(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return NewAPI(catalog.NewService(repo), build.NewService(repo))
}

func validationRequest(model string, required, owned int64) string {
	return fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":1,"owned_quantity":1},{"product_id":%q,"quantity":1,"owned_quantity":0}],"ram_items":[{"ram_model_id":%q,"required_modules":%d,"owned_modules":%d}]}`, activeID, validationBoardID, model, required, owned)
}

func postValidation(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/builds/validate", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Header().Get("Content-Type") != "application/json; charset=utf-8" || !json.Valid(w.Body.Bytes()) {
		t.Fatalf("invalid JSON response: %v %s", w.Header(), w.Body.String())
	}
	return w
}

func TestBuildValidationHTTPRAMSelections(t *testing.T) {
	for _, tt := range []struct {
		name                                           string
		required, owned, total, excess, singles, pairs int64
		pairsOnly                                      bool
	}{
		{"mixed exact", 3, 0, 11500, 0, 1, 1, false},
		{"one module exact", 1, 0, 7500, 0, 1, 0, false},
		{"two module package", 2, 0, 9000, 0, 0, 1, false},
		{"owned modules deducted", 4, 2, 9000, 0, 0, 1, false},
		{"all RAM owned", 2, 2, 5000, 0, 0, 0, false},
		{"least excess ignores inactive single", 3, 0, 13000, 1, 0, 2, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := validationSnapshot()
			if tt.pairsOnly {
				snapshot.Products[2].Product.IsActive = false
			}
			w := postValidation(t, validationAPI(t, snapshot), validationRequest(validationModelID, tt.required, tt.owned))
			var response buildResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			data := response.Data
			if w.Code != 200 || !data.Compatible || data.Issues == nil || len(data.Issues) != 0 || data.TotalPriceTHB != tt.total || len(data.Items) != 2 || len(data.RAMSelections) != 1 {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if data.Items[0].UnitPriceTHB != 25000 || data.Items[0].PurchaseQuantity != 0 || data.Items[0].PurchasePriceTHB != 0 || data.Items[1].PurchasePriceTHB != 5000 {
				t.Fatalf("owned CPU must not be charged: %+v", data.Items)
			}
			selection := data.RAMSelections[0]
			if selection.Packages == nil || selection.ExcessModules != tt.excess || selection.PurchaseModules != tt.required-tt.owned || selection.SuppliedModules != tt.required-tt.owned+tt.excess || selection.TotalPriceTHB != tt.total-5000 {
				t.Fatalf("selection=%+v", selection)
			}
			var singles, pairs int64
			for _, pack := range selection.Packages {
				if pack.ProductID == validationSingleID {
					singles += pack.Quantity
				}
				if pack.ProductID == validationPairID {
					pairs += pack.Quantity
				}
			}
			if singles != tt.singles || pairs != tt.pairs {
				t.Fatalf("packages=%+v", selection.Packages)
			}
		})
	}
}

func TestBuildValidationHTTPCompatibilityIssues(t *testing.T) {
	for _, tt := range []struct {
		name                     string
		cpuMismatch, ramMismatch bool
		owned                    int64
	}{
		{"CPU mismatch", true, false, 0}, {"RAM mismatch", false, true, 0},
		{"both mismatch", true, true, 0}, {"owned RAM mismatch", false, true, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := validationSnapshot()
			if tt.cpuMismatch {
				snapshot.Products[0].Specs.CPUSocket = "LGA1700"
			}
			model := validationModelID
			if tt.ramMismatch {
				model = validationDDR4ModelID
			}
			w := postValidation(t, validationAPI(t, snapshot), validationRequest(model, 2, tt.owned))
			var body buildResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || body.Data.Compatible {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			foundCPU, foundRAM := false, false
			for _, issue := range body.Data.Issues {
				if issue.Message == "" {
					t.Fatal("missing issue message")
				}
				if issue.Code == "CPU_SOCKET_MISMATCH" {
					foundCPU = true
				}
				if issue.Code == "RAM_TYPE_MISMATCH" {
					foundRAM = true
				}
			}
			if foundCPU != tt.cpuMismatch || foundRAM != tt.ramMismatch {
				t.Fatalf("issues=%+v", body.Data.Issues)
			}
		})
	}
}

func TestBuildValidationHTTPInvalidInput(t *testing.T) {
	valid := validationRequest(validationModelID, 3, 0)
	for _, tt := range []struct{ name, body string }{
		{"empty body", ""}, {"null body", "null"}, {"array body", "[]"}, {"missing arrays", "{}"},
		{"null items", `{"items":null,"ram_items":[]}`}, {"missing RAM array", `{"items":[]}`},
		{"missing items array", `{"ram_items":[]}`}, {"null RAM array", `{"items":[],"ram_items":null}`},
		{"bad JSON", "{"}, {"second JSON", valid + "{}"}, {"trailing junk", valid + "oops"},
		{"bad UUID", strings.Replace(valid, activeID, "bad", 1)},
		{"bad RAM UUID", strings.Replace(valid, validationModelID, "bad", 1)},
		{"zero quantity", strings.Replace(valid, `"quantity":1`, `"quantity":0`, 1)},
		{"decimal quantity", strings.Replace(valid, `"quantity":1`, `"quantity":1.5`, 1)},
		{"integer overflow", strings.Replace(valid, `"quantity":1`, `"quantity":9223372036854775807`, 1)},
		{"missing quantity", strings.Replace(valid, `"quantity":1,`, "", 1)},
		{"null quantity", strings.Replace(valid, `"quantity":1`, `"quantity":null`, 1)},
		{"missing owned", strings.Replace(valid, `,"owned_quantity":1`, "", 1)},
		{"owned too many", strings.Replace(valid, `"owned_quantity":1`, `"owned_quantity":2`, 1)},
		{"negative owned", strings.Replace(valid, `"owned_quantity":1`, `"owned_quantity":-1`, 1)},
		{"zero modules", validationRequest(validationModelID, 0, 0)},
		{"owned too many modules", validationRequest(validationModelID, 1, 2)},
		{"missing required modules", strings.Replace(valid, `"required_modules":3,`, "", 1)},
		{"missing owned modules", strings.Replace(valid, `,"owned_modules":0`, "", 1)},
		{"null RAM entry", `{"items":[],"ram_items":[null]}`},
		{"duplicate products", fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":1,"owned_quantity":0},{"product_id":%q,"quantity":1,"owned_quantity":0}],"ram_items":[]}`, activeID, activeID)},
		{"duplicate models", fmt.Sprintf(`{"items":[],"ram_items":[{"ram_model_id":%q,"required_modules":1,"owned_modules":0},{"ram_model_id":%q,"required_modules":1,"owned_modules":0}]}`, validationModelID, validationModelID)},
		{"RAM SKU in items", strings.Replace(valid, activeID, validationSingleID, 1)},
		{"client total", strings.Replace(valid, `{"items":`, `{"total_price_thb":1,"items":`, 1)},
		{"client compatibility", strings.Replace(valid, `{"items":`, `{"compatible":true,"items":`, 1)},
		{"client product price", strings.Replace(valid, `"quantity":1`, `"price_thb":1,"quantity":1`, 1)},
		{"client RAM price", strings.Replace(valid, `"required_modules":3`, `"price_thb":1,"required_modules":3`, 1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := postValidation(t, validationAPI(t, validationSnapshot()), tt.body)
			var body map[string]map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != 400 || len(body) != 1 || len(body["error"]) != 2 || body["error"]["code"] != "VALIDATION_ERROR" || body["error"]["message"] == "" {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestBuildValidationHTTPUnavailable(t *testing.T) {
	for _, tt := range []struct {
		name                         string
		inactiveCPU, inactiveRAM     bool
		unknownProduct, unknownModel bool
	}{
		{"inactive CPU", true, false, false, false}, {"inactive packages", false, true, false, false},
		{"unknown product", false, false, true, false}, {"unknown model", false, false, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := validationSnapshot()
			snapshot.Products[0].Product.IsActive = !tt.inactiveCPU
			if tt.inactiveRAM {
				snapshot.Products[2].Product.IsActive, snapshot.Products[3].Product.IsActive = false, false
			}
			body := validationRequest(validationModelID, 2, 0)
			if tt.unknownProduct {
				body = strings.Replace(body, activeID, inactiveID, 1)
			}
			if tt.unknownModel {
				body = strings.Replace(body, validationModelID, "10000000-0000-0000-0000-000000000099", 1)
			}
			w := postValidation(t, validationAPI(t, snapshot), body)
			if w.Code != 404 || !strings.Contains(w.Body.String(), `"code":"NOT_FOUND"`) {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

type failedBuildRepository struct{}

func (failedBuildRepository) LoadSnapshot(context.Context) (build.Snapshot, error) {
	return build.Snapshot{}, errors.New("internal repository connection detail")
}

func TestBuildValidationHTTPBoundaryErrors(t *testing.T) {
	h := validationAPI(t, validationSnapshot())
	w := postValidation(t, h, strings.Repeat(" ", maxBuildBodyBytes+1)+`{"items":[],"ram_items":[]}`)
	if w.Code != 413 || !strings.Contains(w.Body.String(), `"code":"PAYLOAD_TOO_LARGE"`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	for _, contentType := range []string{"", "text/plain", "application/json;bad"} {
		r := httptest.NewRequest("POST", "/api/v1/builds/validate", strings.NewReader(`{"items":[],"ram_items":[]}`))
		r.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 415 || !json.Valid(w.Body.Bytes()) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
	w = request(t, h, "GET", "/api/v1/builds/validate")
	if w.Code != 405 || w.Header().Get("Allow") != "POST" {
		t.Fatalf("status=%d headers=%v", w.Code, w.Header())
	}
	w = request(t, h, "GET", "/api/v1/builds/validate/")
	if w.Code != 404 {
		t.Fatalf("status=%d", w.Code)
	}
	failed := NewBuildValidation(build.NewService(failedBuildRepository{}))
	w = postValidation(t, failed, `{"items":[],"ram_items":[]}`)
	if w.Code != 500 || strings.Contains(w.Body.String(), "connection") || !strings.Contains(w.Body.String(), `"code":"INTERNAL_ERROR"`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestBuildValidationHTTPEmptyDraftAndCatalogRegression(t *testing.T) {
	h := validationAPI(t, validationSnapshot())
	w := postValidation(t, h, `{"items":[],"ram_items":[]}`)
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"data":{"compatible":true,"issues":[],"items":[],"ram_selections":[],"total_price_thb":0}}` {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	w = request(t, h, "GET", "/api/v1/products/"+activeID)
	var body struct {
		Data struct {
			Product json.RawMessage `json:"product"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	assertProductContract(t, body.Data.Product)
}
