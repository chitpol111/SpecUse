package handler

import (
	"context"
	"net/http"
	"time"

	"specuse/backend/internal/account"
)

func WithHealth(next http.Handler, mode string, probe func(context.Context) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/health" {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if !allowedMethod(w, r, "GET") {
			return
		}
		if probe != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := probe(ctx); err != nil {
				accountError(w, account.ErrUnavailable)
				return
			}
		}
		writeJSON(w, 200, map[string]any{"data": map[string]string{"status": "ok", "catalog_backend": mode}})
	})
}
