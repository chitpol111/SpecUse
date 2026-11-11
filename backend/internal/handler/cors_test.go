package handler

import (
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	for _, tt := range []struct {
		name       string
		origins    []string
		origin     string
		method     string
		preflight  string
		wantOrigin string
		status     int
	}{
		{"allowed", []string{"http://localhost:3000"}, "http://localhost:3000", "GET", "", "http://localhost:3000", 200},
		{"denied", []string{"http://localhost:3000"}, "https://other.example", "GET", "", "", 200},
		{"unset", nil, "http://localhost:3000", "GET", "", "", 200},
		{"wildcard ignored", []string{"*"}, "https://other.example", "GET", "", "", 200},
		{"preflight", []string{"http://localhost:3000"}, "http://localhost:3000", "OPTIONS", "GET", "http://localhost:3000", 204},
		{"preflight denied", nil, "https://other.example", "OPTIONS", "GET", "", 405},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := WithCORS(testHandler(t), tt.origins)
			r := httptest.NewRequest(tt.method, "/api/v1/products", nil)
			r.Header.Set("Origin", tt.origin)
			r.Header.Set("Access-Control-Request-Method", tt.preflight)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tt.status || w.Header().Get("Access-Control-Allow-Origin") != tt.wantOrigin {
				t.Fatalf("status=%d headers=%v", w.Code, w.Header())
			}
			if w.Header().Get("Vary") == "" {
				t.Fatal("Vary: Origin missing")
			}
			if tt.status == 204 && (w.Body.Len() != 0 || w.Header().Get("Access-Control-Allow-Methods") != "GET, POST") {
				t.Fatal("incorrect preflight response")
			}
		})
	}
}
