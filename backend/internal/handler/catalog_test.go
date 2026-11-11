package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"specuse/backend/internal/catalog"
	"specuse/backend/internal/repository"
)

const activeID = "abcdefab-0000-0000-0000-000000000001"
const inactiveID = "abcdefab-0000-0000-0000-000000000002"

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	repo, err := repository.NewMemory([]catalog.Product{
		{ID: activeID, Name: "Test CPU", Category: "cpu", Brand: "Example", PriceTHB: 25000, IsActive: true},
		{ID: inactiveID, Name: "Hidden CPU", Category: "cpu", Brand: "Example", PriceTHB: 99, IsActive: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	return NewCatalog(catalog.NewService(repo))
}

func request(t *testing.T, h http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, target, nil))
	if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", w.Header().Get("Content-Type"))
	}
	if !json.Valid(w.Body.Bytes()) {
		t.Fatalf("invalid JSON: %s", w.Body.String())
	}
	return w
}

func assertProductContract(t *testing.T, raw json.RawMessage) {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) < 6 || len(fields) > 8 {
		t.Fatalf("expected six original fields and optional compatibility metadata, got %s", raw)
	}
	for key := range fields {
		switch key {
		case "id", "name", "category", "brand", "price_thb", "is_active", "specs", "ram_model_id":
		default:
			t.Fatalf("unexpected product field %s", key)
		}
	}
	for _, key := range []string{"id", "name", "category", "brand", "price_thb", "is_active"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("missing field %s", key)
		}
	}
	var price int64
	var active bool
	if err := json.Unmarshal(fields["price_thb"], &price); err != nil || price != 25000 {
		t.Fatalf("price must be an integer in THB: %s", fields["price_thb"])
	}
	if err := json.Unmarshal(fields["is_active"], &active); err != nil || !active {
		t.Fatal("inactive product returned")
	}
}

func TestListHTTPContract(t *testing.T) {
	h := testHandler(t)
	for _, query := range []string{"", "?category=cpu&search=test&page=1&page_size=20", "?category=%20cpu%20&search=%20TEST%20"} {
		w := request(t, h, http.MethodGet, "/api/v1/products"+query)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var body struct {
			Data struct {
				Items    []json.RawMessage `json:"items"`
				Page     int               `json:"page"`
				PageSize int               `json:"page_size"`
				Total    int               `json:"total"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Data.Page != 1 || body.Data.PageSize != 20 || body.Data.Total != 1 || len(body.Data.Items) != 1 {
			t.Fatalf("unexpected body: %s", w.Body.String())
		}
		assertProductContract(t, body.Data.Items[0])
	}
}

func TestEmptyPageHTTPContract(t *testing.T) {
	for _, query := range []string{"?page=2&page_size=1", "?category=unknown", "?search=hidden"} {
		w := request(t, testHandler(t), http.MethodGet, "/api/v1/products"+query)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"items":[]`) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
	repo, _ := repository.NewMemory(nil)
	w := request(t, NewCatalog(catalog.NewService(repo)), http.MethodGet, "/api/v1/products")
	if strings.TrimSpace(w.Body.String()) != `{"data":{"items":[],"page":1,"page_size":20,"total":0}}` {
		t.Fatalf("unexpected empty catalog: %s", w.Body.String())
	}
}

func TestDetailHTTPContract(t *testing.T) {
	w := request(t, testHandler(t), http.MethodGet, "/api/v1/products/"+strings.ToUpper(activeID))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
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

func TestHTTPErrorContract(t *testing.T) {
	tests := []struct {
		method string
		target string
		status int
		code   string
	}{
		{"GET", "/api/v1/products/not-a-uuid", 400, "VALIDATION_ERROR"},
		{"GET", "/api/v1/products/" + inactiveID, 404, "NOT_FOUND"},
		{"GET", "/api/v1/products/abcdefab-0000-0000-0000-000000000099", 404, "NOT_FOUND"},
		{"POST", "/api/v1/products", 405, "METHOD_NOT_ALLOWED"},
		{"DELETE", "/api/v1/products/" + activeID, 405, "METHOD_NOT_ALLOWED"},
		{"GET", "/missing", 404, "NOT_FOUND"},
		{"GET", "/api/v1/products/", 404, "NOT_FOUND"},
		{"GET", "/api/v1/products/" + activeID + "/extra", 404, "NOT_FOUND"},
	}
	for _, query := range []string{
		"page=0", "page=-1", "page=abc", "page=", "page=1&page=2", "page_size=0", "page_size=-1", "page_size=1.5", "page_size=", "page_size=1&page_size=2",
		"page=9999999999999999999999999999", "page=9223372036854775807&page_size=20", "category=cpu&category=ram", "search=a&search=b", "search=bad%zz", "search=a;b",
	} {
		tests = append(tests, struct {
			method string
			target string
			status int
			code   string
		}{"GET", "/api/v1/products?" + query, 400, "VALIDATION_ERROR"})
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.target, func(t *testing.T) {
			w := request(t, testHandler(t), tt.method, tt.target)
			var body map[string]map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != tt.status || len(body) != 1 || len(body["error"]) != 2 || body["error"]["code"] != tt.code || body["error"]["message"] == "" {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if tt.status == 405 && w.Header().Get("Allow") != "GET" {
				t.Fatal("405 missing Allow header")
			}
		})
	}
}

type failedRepository struct{}

func (failedRepository) ListActive(context.Context, catalog.Filter) ([]catalog.ProductDetails, int, error) {
	return nil, 0, errors.New("private database credential")
}

func (failedRepository) FindActiveByID(context.Context, string) (catalog.ProductDetails, error) {
	return catalog.ProductDetails{}, errors.New("private database credential")
}

func TestInternalErrorsAreSanitized(t *testing.T) {
	h := NewCatalog(catalog.NewService(failedRepository{}))
	for _, target := range []string{"/api/v1/products", "/api/v1/products/" + activeID} {
		w := request(t, h, http.MethodGet, target)
		if w.Code != 500 || strings.Contains(w.Body.String(), "credential") || !strings.Contains(w.Body.String(), `"code":"INTERNAL_ERROR"`) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
}
