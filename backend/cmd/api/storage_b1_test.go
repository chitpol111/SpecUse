package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/handler"
)

func storageB1API(t *testing.T) http.Handler {
	t.Helper()
	repo, err := loadCatalog("local", "../../testdata/catalog.storage-b1.synthetic.local.json")
	if err != nil {
		t.Fatal(err)
	}
	return handler.WithCORS(handler.NewAPI(catalog.NewService(repo), build.NewService(repo)), []string{"http://localhost:3000", "http://127.0.0.1:3000"})
}
func storageB1Example(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("../../examples/storage-b1/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func TestStorageB1RunnableExamplesAndCoverage(t *testing.T) {
	h := storageB1API(t)
	for _, tt := range []struct {
		name, overall, storage string
		total                  float64
	}{
		{"validate", "unknown", "passed", 1000}, {"validate-multi-sku", "unknown", "passed", 2500},
		{"validate-too-many-ports", "failed", "failed", 1000}, {"validate-no-ports", "failed", "failed", 1000},
		{"validate-null-ports", "unknown", "unknown", 1000}, {"validate-sharing", "unknown", "unknown", 1000},
		{"validate-bios", "unknown", "unknown", 1000}, {"validate-evidence-missing", "unknown", "unknown", 1000},
		{"validate-gpu", "unknown", "unknown", 1000}, {"validate-m2", "unknown", "unknown", 1000},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := performJSON(h, "POST", extendedPath, storageB1Example(t, tt.name+".request.json"))
			body := decodeMap(t, w.Body.String())
			want := decodeMap(t, storageB1Example(t, tt.name+".response.json"))
			if w.Code != 200 || !reflect.DeepEqual(body, want) {
				t.Fatalf("example drift %d %s", w.Code, w.Body.String())
			}
			d := body["data"].(map[string]any)
			c := d["compatibility"].(map[string]any)
			if d["compatible"] != false || d["total_price_thb"] != tt.total || c["mode"] != build.GPUStorageMode || c["status"] != tt.overall || c["build_completeness"] != "not_evaluated" {
				t.Fatal(w.Body.String())
			}
			checks := map[string]map[string]any{}
			for _, v := range c["checks"].([]any) {
				check := v.(map[string]any)
				checks[check["code"].(string)] = check
				for _, key := range []string{"missing_fields", "unsupported_fields"} {
					refs, ok := check[key].([]any)
					if !ok {
						t.Fatalf("nullable or wrong shape %s", key)
					}
					for _, v := range refs {
						ref := v.(map[string]any)
						if len(ref) != 4 || ref["location"] == nil || ref["path"] == nil {
							t.Fatal("FieldReference schema changed")
						}
					}
				}
			}
			if checks["STORAGE_MOTHERBOARD"]["status"] != tt.storage || checks["MOTHERBOARD_RESOURCES"]["status"] != tt.storage || checks["SYSTEM_PSU"]["status"] != "unknown" || checks["SYSTEM_CASE"]["status"] != "unknown" {
				t.Fatal(w.Body.String())
			}
			if tt.overall == "failed" && len(d["issues"].([]any)) == 0 {
				t.Fatal("missing proven failure issue")
			}
			if tt.name == "validate" {
				rows := d["items"].([]any)
				row := rows[2].(map[string]any)
				if row["quantity"] != float64(3) || row["owned_quantity"] != float64(2) || row["purchase_quantity"] != float64(1) || row["purchase_price_thb"] != float64(1000) {
					t.Fatal("owned pricing/counts changed")
				}
			}
		})
	}
	for _, tt := range []struct{ name, path string }{
		{"products-storage", "/api/v1/products?category=storage"}, {"product-motherboard", "/api/v1/products/94000000-0000-4000-8000-000000000002"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := performJSON(h, "GET", tt.path, "")
			if w.Code != 200 || !reflect.DeepEqual(decodeMap(t, w.Body.String()), decodeMap(t, storageB1Example(t, tt.name+".response.json"))) {
				t.Fatal("GET example drift")
			}
		})
	}
}
func TestStorageB1HTTPPermutationAndLegacy11500(t *testing.T) {
	h := storageB1API(t)
	body := storageB1Example(t, "validate-multi-sku.request.json")
	first := decodeMap(t, performJSON(h, "POST", extendedPath, body).Body.String())["data"].(map[string]any)
	var request struct {
		Items    []map[string]any `json:"items"`
		RAMItems []any            `json:"ram_items"`
	}
	if err := json.Unmarshal([]byte(body), &request); err != nil {
		t.Fatal(err)
	}
	for i, j := 0, len(request.Items)-1; i < j; i, j = i+1, j-1 {
		request.Items[i], request.Items[j] = request.Items[j], request.Items[i]
	}
	encoded, _ := json.Marshal(request)
	second := decodeMap(t, performJSON(h, "POST", extendedPath, string(encoded)).Body.String())["data"].(map[string]any)
	for _, key := range []string{"compatible", "compatibility", "issues", "total_price_thb"} {
		if !reflect.DeepEqual(first[key], second[key]) {
			t.Fatalf("permutation affected %s", key)
		}
	}
	bodyBytes, err := os.ReadFile("../../examples/validate.request.json")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("../../examples/validate.response.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/v1/builds/validate", extendedPath} {
		w := performJSON(h, "POST", path, string(bodyBytes))
		d := decodeMap(t, w.Body.String())["data"].(map[string]any)
		if w.Code != 200 || d["total_price_thb"] != float64(11500) {
			t.Fatal("11500 regression")
		}
		if path == extendedPath {
			if d["compatibility"].(map[string]any)["status"] != "unknown" {
				t.Fatal("PSU/Case unknown lost")
			}
		} else if !reflect.DeepEqual(decodeMap(t, w.Body.String()), decodeMap(t, string(want))) {
			t.Fatal("legacy response changed")
		}
	}
	// Existing catalog and price metadata remain authoritative.
	w := performJSON(h, "GET", "/api/v1/products/91000000-0000-4000-8000-000000000006", "")
	p := decodeMap(t, w.Body.String())["data"].(map[string]any)["product"].(map[string]any)
	if p["price_thb"] != float64(1000) || p["specs"].(map[string]any)["host"].(map[string]any)["connector"] != "sata_data" {
		t.Fatal("catalog/validate data drift")
	}
}
func TestStorageB1LimitsNotFoundAndNoClientPortContext(t *testing.T) {
	h := storageB1API(t)
	valid := storageB1Example(t, "validate.request.json")
	for _, tt := range []struct {
		name, body string
		status     int
		code       string
	}{
		{"storage limit", strings.Replace(valid, `"quantity": 3`, `"quantity": 9`, 1), 400, "UNSUPPORTED_BUILD_SCOPE"},
		{"unknown before limit", `{"items":[{"product_id":"91000000-0000-4000-8000-000000000006","quantity":9,"owned_quantity":9},{"product_id":"94000000-0000-4000-8000-000000000999","quantity":1,"owned_quantity":0}],"ram_items":[]}`, 404, "NOT_FOUND"},
		{"inactive owned", `{"items":[{"product_id":"abcdefab-0000-0000-0000-000000000002","quantity":1,"owned_quantity":1}],"ram_items":[]}`, 404, "NOT_FOUND"},
		{"GPU limit", `{"items":[{"product_id":"91000000-0000-4000-8000-000000000004","quantity":2,"owned_quantity":2}],"ram_items":[]}`, 400, "UNSUPPORTED_BUILD_SCOPE"},
		{"client port", strings.Replace(valid, `"owned_quantity": 2`, `"owned_quantity": 2,"port_id":"SATA_1"`, 1), 400, "VALIDATION_ERROR"},
		{"client BIOS", strings.Replace(valid, `"ram_items": []`, `"ram_items": [],"bios_version":"guessed"`, 1), 400, "VALIDATION_ERROR"},
		{"client price", strings.Replace(valid, `"owned_quantity": 2`, `"owned_quantity": 2,"price_thb":1`, 1), 400, "VALIDATION_ERROR"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := performJSON(h, "POST", extendedPath, tt.body)
			e := decodeMap(t, w.Body.String())["error"].(map[string]any)
			if w.Code != tt.status || e["code"] != tt.code || len(e) != 2 {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
		})
	}
	for _, origin := range []string{"http://localhost:3000", "http://127.0.0.1:3000"} {
		for _, method := range []string{"POST", "OPTIONS"} {
			t.Run(origin+method, func(t *testing.T) {
				r := httptest.NewRequest(method, extendedPath, strings.NewReader(valid))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Origin", origin)
				r.Header.Set("Access-Control-Request-Method", "POST")
				r.Header.Set("Access-Control-Request-Headers", "content-type")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				want := 200
				if method == "OPTIONS" {
					want = 204
				}
				if w.Code != want || w.Header().Get("Access-Control-Allow-Origin") != origin {
					t.Fatal("CORS regression")
				}
			})
		}
	}
}
func TestStorageB1FixtureIsSyntheticAndLocalOnly(t *testing.T) {
	path := "../../testdata/catalog.storage-b1.synthetic.local.json"
	for _, env := range []string{"", "production", "staging"} {
		if _, err := loadCatalog(env, path); err == nil {
			t.Fatal("non-local fixture accepted")
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var products []catalog.ProductDetails
	if err := json.Unmarshal(data, &products); err != nil {
		t.Fatal(err)
	}
	if len(products) != 28 {
		t.Fatal("unexpected fixture size")
	}
	for _, p := range products {
		if strings.HasPrefix(p.ID, "94000000-") {
			if !strings.HasPrefix(p.Name, "[SYNTHETIC FIXTURE ONLY]") {
				t.Fatal("unlabelled synthetic product")
			}
		}
	}
	for _, word := range []string{"ASUS", "Kingston", "PRIME B550", "SA400S37"} {
		if strings.Contains(string(data), word) {
			t.Fatal("manufacturer product/evidence inserted into synthetic data")
		}
	}
}
