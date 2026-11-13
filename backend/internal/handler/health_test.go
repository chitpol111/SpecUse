package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthUsesReadinessProbeAndMasksErrors(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want int
	}{
		{"ready", nil, 200}, {"database down", errors.New("connection SECRET must not leak"), 503},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			h := WithHealth(http.NotFoundHandler(), "supabase", func(ctx context.Context) error {
				calls++
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("missing probe deadline")
				}
				return tt.err
			})
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/health", nil))
			if w.Code != tt.want || calls != 1 || strings.Contains(w.Body.String(), "SECRET") {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
		})
	}
	h := WithHealth(http.NotFoundHandler(), "memory", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/health", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"catalog_backend":"memory"`) {
		t.Fatal("memory mode must be explicit")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/health", nil))
	if w.Code != 405 {
		t.Fatal("method not protected")
	}
}
