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

func fixtureAPI(t *testing.T) http.Handler {
	t.Helper()
	repo, err := loadCatalog("local", "../../testdata/catalog.local.json")
	if err != nil {
		t.Fatal(err)
	}
	return handler.WithCORS(handler.NewAPI(catalog.NewService(repo), build.NewService(repo)),
		[]string{"http://localhost:3000", "http://127.0.0.1:3000"})
}

// These are the exact request/response files delivered to the frontend.
func TestFrontendExamplesMatchLocalFixtureAPI(t *testing.T) {
	h := fixtureAPI(t)
	for _, example := range []string{"validate", "validate-incompatible", "validate-excess"} {
		t.Run(example, func(t *testing.T) {
			request, err := os.ReadFile("../../examples/" + example + ".request.json")
			if err != nil {
				t.Fatal(err)
			}
			expected, err := os.ReadFile("../../examples/" + example + ".response.json")
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("POST", "/api/v1/builds/validate", strings.NewReader(string(request)))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 200 {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			var want, got any
			if err := json.Unmarshal(expected, &want); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("frontend example drifted:\nwant=%s\ngot=%s", expected, w.Body.String())
			}
		})
	}
}

func TestFrontendOriginsCanPostAndPreflight(t *testing.T) {
	h := fixtureAPI(t)
	for _, origin := range []string{"http://localhost:3000", "http://127.0.0.1:3000"} {
		t.Run(origin, func(t *testing.T) {
			for _, method := range []string{"POST", "OPTIONS"} {
				r := httptest.NewRequest(method, "/api/v1/builds/validate", strings.NewReader(`{"items":[],"ram_items":[]}`))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Origin", origin)
				if method == "OPTIONS" {
					r.Header.Set("Access-Control-Request-Method", "POST")
					r.Header.Set("Access-Control-Request-Headers", "content-type")
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				wantStatus := 200
				if method == "OPTIONS" {
					wantStatus = 204
				}
				if w.Code != wantStatus || w.Header().Get("Access-Control-Allow-Origin") != origin {
					t.Fatalf("method=%s status=%d headers=%v", method, w.Code, w.Header())
				}
				if method == "OPTIONS" && (w.Header().Get("Access-Control-Allow-Methods") != "GET, POST" || w.Body.Len() != 0) {
					t.Fatalf("invalid POST preflight: %v", w.Header())
				}
			}
		})
	}
}

func TestFixtureCatalogRetainsOriginalFieldsAndAddsTrustedSpecs(t *testing.T) {
	h := fixtureAPI(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/products", nil))
	var body struct {
		Data struct {
			Items []map[string]any `json:"items"`
			Total int              `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || body.Data.Total != 7 || len(body.Data.Items) != 7 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	fixture, err := os.ReadFile("../../testdata/catalog.local.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected []map[string]any
	if err := json.Unmarshal(fixture, &expected); err != nil {
		t.Fatal(err)
	}
	byID := map[string]map[string]any{}
	for _, product := range expected {
		byID[product["id"].(string)] = product
	}
	for _, product := range body.Data.Items {
		if product["is_active"] != true {
			t.Fatalf("catalog response changed: %+v", product)
		}
		for _, name := range []string{"id", "name", "category", "brand", "price_thb", "is_active"} {
			if _, exists := product[name]; !exists {
				t.Fatalf("missing %s", name)
			}
		}
		if !reflect.DeepEqual(product, byID[product["id"].(string)]) {
			t.Fatalf("public fields/specs must match trusted catalog data: %+v", product)
		}
	}
}

func TestCatalogExamplesMatchLocalFixtureAPI(t *testing.T) {
	h := fixtureAPI(t)
	for _, example := range []struct{ name, path string }{
		{"products", "/api/v1/products"},
		{"product-cpu", "/api/v1/products/abcdefab-0000-0000-0000-000000000001"},
		{"product-motherboard", "/api/v1/products/abcdefab-0000-0000-0000-000000000004"},
		{"product-ram-single", "/api/v1/products/abcdefab-0000-0000-0000-000000000005"},
		{"product-ram-pair", "/api/v1/products/abcdefab-0000-0000-0000-000000000006"},
	} {
		t.Run(example.name, func(t *testing.T) {
			expected, err := os.ReadFile("../../examples/" + example.name + ".response.json")
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", example.path, nil))
			var want, got any
			if err := json.Unmarshal(expected, &want); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || !reflect.DeepEqual(want, got) {
				t.Fatalf("status=%d want=%s got=%s", w.Code, expected, w.Body.String())
			}
		})
	}
}

func TestCatalogMetadataSurvivesPaginationAndInactiveFiltering(t *testing.T) {
	h := fixtureAPI(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/products?category=ram&page=2&page_size=1", nil))
	var page struct {
		Data catalog.Page `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || page.Data.Total != 4 || page.Data.Page != 2 || len(page.Data.Items) != 1 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	product := page.Data.Items[0]
	if product.ID != "abcdefab-0000-0000-0000-000000000006" || product.Specs == nil || product.Specs.ModulesPerPackage != 2 || product.RAMModelID == "" {
		t.Fatalf("metadata lost during pagination: %+v", product)
	}
	for _, id := range []string{"abcdefab-0000-0000-0000-000000000002", "abcdefab-0000-0000-0000-000000000009"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/products/"+id, nil))
		if w.Code != 404 || strings.Contains(w.Body.String(), `"specs"`) {
			t.Fatalf("inactive metadata leaked: %d %s", w.Code, w.Body.String())
		}
	}
}
