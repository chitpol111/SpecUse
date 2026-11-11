package handler

import (
	"net/http"

	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
)

// NewAPI keeps JSON errors on unknown routes rather than net/http's text errors.
func NewAPI(catalogService *catalog.Service, buildService *build.Service) http.Handler {
	products := NewCatalog(catalogService)
	validation := NewBuildValidation(buildService)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/builds/validate" {
			validation.ServeHTTP(w, r)
			return
		}
		products.ServeHTTP(w, r)
	})
}
