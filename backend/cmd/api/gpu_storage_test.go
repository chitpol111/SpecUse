package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/handler"
)

const extendedPath = "/api/v1/builds/validate?validation_mode=gpu_storage_v1"

func phaseAAPI(t *testing.T) http.Handler {
	t.Helper()
	repo, err := loadCatalog("local", "../../testdata/catalog.gpu-storage.synthetic.local.json")
	if err != nil {
		t.Fatal(err)
	}
	return handler.WithCORS(handler.NewAPI(catalog.NewService(repo), build.NewService(repo)), []string{"http://localhost:3000", "http://127.0.0.1:3000"})
}

func performJSON(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func phaseAExample(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("../../examples/gpu-storage-a/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func decodeMap(t *testing.T, body string) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestPhaseACatalogMetadataAndSharedPrices(t *testing.T) {
	h := phaseAAPI(t)
	w := performJSON(h, "GET", "/api/v1/products", "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var page struct {
		Data catalog.Page `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Data.Total != 19 || len(page.Data.Items) != 19 {
		t.Fatalf("unexpected combined fixture: %+v", page.Data)
	}
	for _, p := range page.Data.Items {
		detail := performJSON(h, "GET", "/api/v1/products/"+p.ID, "")
		var response struct {
			Data struct {
				Product catalog.ProductDetails `json:"product"`
			} `json:"data"`
		}
		if err := json.Unmarshal(detail.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if detail.Code != 200 || !reflect.DeepEqual(p, response.Data.Product) {
			t.Fatalf("list/detail drift for %s", p.ID)
		}
		if p.Category == "gpu" {
			if p.Specs == nil || p.Specs.GPU == nil {
				t.Fatalf("GPU specs missing for %s", p.ID)
			}
			if p.ID == "91000000-0000-4000-8000-000000000004" && *p.Specs.GPU.Power.BoardPowerW != 120 {
				t.Fatal("GPU typed metadata changed")
			}
		}
		if p.Category == "storage" && (p.Specs == nil || p.Specs.Storage == nil) {
			t.Fatalf("Storage specs missing %s", p.ID)
		}
		if p.Category == "motherboard" && p.Specs.Motherboard == nil {
			t.Fatal("motherboard metadata not nullable typed")
		}
	}
	w = performJSON(h, "POST", extendedPath, phaseAExample(t, "validate.request.json"))
	d := decodeMap(t, w.Body.String())["data"].(map[string]any)
	if w.Code != 200 || d["total_price_thb"] != float64(8000) || d["compatible"] != false {
		t.Fatalf("bad result %s", w.Body.String())
	}
	byID := map[string]catalog.ProductDetails{}
	for _, p := range page.Data.Items {
		byID[p.ID] = p
	}
	for _, v := range d["items"].([]any) {
		row := v.(map[string]any)
		p := byID[row["product_id"].(string)]
		if row["unit_price_thb"] != float64(p.PriceTHB) {
			t.Fatal("GET/POST price mismatch")
		}
		if row["purchase_quantity"] != row["quantity"].(float64)-row["owned_quantity"].(float64) {
			t.Fatal("owned counted twice")
		}
	}
	c := d["compatibility"].(map[string]any)
	if c["mode"] != build.GPUStorageMode || c["status"] != "unknown" {
		t.Fatal(w.Body.String())
	}
	for _, v := range c["checks"].([]any) {
		check := v.(map[string]any)
		if check["scope"] != "basic" && check["status"] != "unknown" {
			t.Fatalf("stage B/C ran: %+v", check)
		}
	}
	// Deliberately replace Board A with SATA-only Board B: stage A must NOT
	// execute the phase B NVMe/port mismatch rule from the proposal examples.
	body := strings.Replace(phaseAExample(t, "validate.request.json"), "91000000-0000-4000-8000-000000000002", "91000000-0000-4000-8000-000000000007", 1)
	w = performJSON(h, "POST", extendedPath, body)
	d = decodeMap(t, w.Body.String())["data"].(map[string]any)
	if w.Code != 200 || d["compatibility"].(map[string]any)["status"] != "unknown" || len(d["issues"].([]any)) != 0 {
		t.Fatalf("out-of-scope hardware check: %s", w.Body.String())
	}
}

func TestPhaseALegacyResponseAnd11500Regression(t *testing.T) {
	h := phaseAAPI(t)
	for _, example := range []string{"validate", "validate-incompatible", "validate-excess"} {
		t.Run(example, func(t *testing.T) {
			body, err := os.ReadFile("../../examples/" + example + ".request.json")
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile("../../examples/" + example + ".response.json")
			if err != nil {
				t.Fatal(err)
			}
			w := performJSON(h, "POST", "/api/v1/builds/validate", string(body))
			if w.Code != 200 || !reflect.DeepEqual(decodeMap(t, w.Body.String()), decodeMap(t, string(want))) {
				t.Fatalf("legacy changed %s", w.Body.String())
			}
			w = performJSON(h, "POST", extendedPath, string(body))
			d := decodeMap(t, w.Body.String())["data"].(map[string]any)
			status := "unknown"
			if example == "validate-incompatible" {
				status = "failed"
			}
			if w.Code != 200 || d["compatibility"].(map[string]any)["status"] != status || d["compatible"] != false {
				t.Fatal(w.Body.String())
			}
			if example == "validate" && d["total_price_thb"] != float64(11500) {
				t.Fatal("existing 11500 example changed")
			}
		})
	}
}

func TestPhaseAHTTPInputErrorsAndScope(t *testing.T) {
	h := phaseAAPI(t)
	valid := `{"items":[],"ram_items":[]}`
	for _, tt := range []struct {
		name, path, body string
		status           int
		code             string
	}{
		{"unknown mode", "/api/v1/builds/validate?validation_mode=other", valid, 400, "UNSUPPORTED_VALIDATION_MODE"},
		{"empty mode", "/api/v1/builds/validate?validation_mode=", valid, 400, "UNSUPPORTED_VALIDATION_MODE"},
		{"duplicate mode", extendedPath + "&validation_mode=gpu_storage_v1", valid, 400, "VALIDATION_ERROR"},
		{"malformed query", "/api/v1/builds/validate?validation_mode=%zz", valid, 400, "VALIDATION_ERROR"},
		{"missing arrays", extendedPath, `{}`, 400, "VALIDATION_ERROR"},
		{"null array", extendedPath, `{"items":null,"ram_items":[]}`, 400, "VALIDATION_ERROR"},
		{"null body", extendedPath, `null`, 400, "VALIDATION_ERROR"},
		{"price from client", extendedPath, `{"items":[],"ram_items":[],"total_price_thb":1}`, 400, "VALIDATION_ERROR"},
		{"specs from client", extendedPath, `{"items":[],"ram_items":[],"specs":{"cpu_socket":"guessed"}}`, 400, "VALIDATION_ERROR"},
		{"status from client", extendedPath, `{"items":[],"ram_items":[],"compatibility":{"status":"passed"}}`, 400, "VALIDATION_ERROR"},
		{"mode in body", extendedPath, `{"items":[],"ram_items":[],"validation_mode":"gpu_storage_v1"}`, 400, "VALIDATION_ERROR"},
		{"invalid UUID", extendedPath, `{"items":[{"product_id":"bad","quantity":1,"owned_quantity":0}],"ram_items":[]}`, 400, "VALIDATION_ERROR"},
		{"missing quantity", extendedPath, `{"items":[{"product_id":"91000000-0000-4000-8000-000000000004","owned_quantity":0}],"ram_items":[]}`, 400, "VALIDATION_ERROR"},
		{"fractional quantity", extendedPath, `{"items":[{"product_id":"91000000-0000-4000-8000-000000000004","quantity":1.5,"owned_quantity":0}],"ram_items":[]}`, 400, "VALIDATION_ERROR"},
		{"zero quantity", extendedPath, `{"items":[{"product_id":"91000000-0000-4000-8000-000000000004","quantity":0,"owned_quantity":0}],"ram_items":[]}`, 400, "VALIDATION_ERROR"},
		{"owned greater than required", extendedPath, `{"items":[{"product_id":"91000000-0000-4000-8000-000000000004","quantity":1,"owned_quantity":2}],"ram_items":[]}`, 400, "VALIDATION_ERROR"},
		{"GPU limit", extendedPath, phaseAExample(t, "validate-gpu-limit.request.json"), 400, "UNSUPPORTED_BUILD_SCOPE"},
		{"GPU limit across two SKUs", extendedPath, `{"items":[{"product_id":"91000000-0000-4000-8000-000000000004","quantity":1,"owned_quantity":1},{"product_id":"91000000-0000-4000-8000-000000000008","quantity":1,"owned_quantity":1}],"ram_items":[]}`, 400, "UNSUPPORTED_BUILD_SCOPE"},
		{"unsupported Storage limit", extendedPath, phaseAExample(t, "validate-storage-limit.request.json"), 400, "UNSUPPORTED_BUILD_SCOPE"},
		{"unknown product before limit", extendedPath, phaseAExample(t, "validate-not-found.request.json"), 404, "NOT_FOUND"},
		{"inactive fully owned", extendedPath, `{"items":[{"product_id":"abcdefab-0000-0000-0000-000000000002","quantity":1,"owned_quantity":1}],"ram_items":[]}`, 404, "NOT_FOUND"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := performJSON(h, "POST", tt.path, tt.body)
			e := decodeMap(t, w.Body.String())["error"].(map[string]any)
			if w.Code != tt.status || e["code"] != tt.code || len(e) != 2 {
				t.Fatalf("got %d %s", w.Code, w.Body.String())
			}
		})
	}
	for _, tt := range []struct {
		name, method, contentType, body string
		want                            int
	}{
		{"method", "GET", "application/json", valid, 405},
		{"media", "POST", "text/plain", valid, 415},
		{"large body", "POST", "application/json", strings.Repeat(" ", 1<<20) + valid, 413},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, extendedPath, strings.NewReader(tt.body))
			r.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
			if tt.want == 405 && w.Header().Get("Allow") != "POST" {
				t.Fatal("missing Allow")
			}
		})
	}
}

func TestPhaseACORSForBothFrontendOrigins(t *testing.T) {
	h := phaseAAPI(t)
	for _, origin := range []string{"http://localhost:3000", "http://127.0.0.1:3000"} {
		for _, method := range []string{"POST", "OPTIONS"} {
			t.Run(origin+method, func(t *testing.T) {
				r := httptest.NewRequest(method, extendedPath, strings.NewReader(`{"items":[],"ram_items":[]}`))
				r.Header.Set("Origin", origin)
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Access-Control-Request-Method", "POST")
				r.Header.Set("Access-Control-Request-Headers", "content-type")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				want := 200
				if method == "OPTIONS" {
					want = 204
				}
				if w.Code != want || w.Header().Get("Access-Control-Allow-Origin") != origin {
					t.Fatalf("%d %v", w.Code, w.Header())
				}
			})
		}
	}
}

func TestPhaseARunnableExamplesMatchActualAPI(t *testing.T) {
	h := phaseAAPI(t)
	for _, name := range []string{"validate", "validate-unsupported", "validate-cpu-conflict", "validate-ram-conflict", "validate-storage-limit", "validate-gpu-limit", "validate-not-found", "validate-null-metadata"} {
		t.Run(name, func(t *testing.T) {
			w := performJSON(h, "POST", extendedPath, phaseAExample(t, name+".request.json"))
			want := decodeMap(t, phaseAExample(t, name+".response.json"))
			got := decodeMap(t, w.Body.String())
			status := 200
			if name == "validate-storage-limit" || name == "validate-gpu-limit" {
				status = 400
			}
			if name == "validate-not-found" {
				status = 404
			}
			if w.Code != status || !reflect.DeepEqual(want, got) {
				t.Fatalf("example %s drifted: %d %s", name, w.Code, w.Body.String())
			}
		})
	}
	for _, tt := range []struct{ name, path string }{
		{"products-gpu", "/api/v1/products?category=gpu"},
		{"products-storage", "/api/v1/products?category=storage"},
		{"product-motherboard", "/api/v1/products/91000000-0000-4000-8000-000000000002"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := performJSON(h, "GET", tt.path, "")
			if w.Code != 200 || !reflect.DeepEqual(decodeMap(t, w.Body.String()), decodeMap(t, phaseAExample(t, tt.name+".response.json"))) {
				t.Fatal(w.Body.String())
			}
		})
	}
}

func TestPhaseAFixtureRejectsUnknownNestedFields(t *testing.T) {
	body, err := os.ReadFile("../../testdata/catalog.gpu-storage.synthetic.local.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ name, from, to string }{
		{"GPU unknown key", `"chipset_name": "SYNTHETIC-GPU-A"`, `"chipset_name": "SYNTHETIC-GPU-A", "guessed_field": 1`},
		{"GPU wrong enum", `"connector": "pcie_x16"`, `"connector": "guessed"`},
		{"unknown evidence field", `"source_id": "synthetic-v1"`, `"source_id": "synthetic-v1", "secret_field": "not-supported"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			changed := strings.Replace(string(body), tt.from, tt.to, 1)
			if changed == string(body) {
				t.Fatal("fixture test target missing")
			}
			path := filepath.Join(t.TempDir(), "synthetic.json")
			if err := os.WriteFile(path, []byte(changed), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := loadCatalog("local", path); err == nil {
				t.Fatal("bad nested metadata accepted")
			}
		})
	}
}
