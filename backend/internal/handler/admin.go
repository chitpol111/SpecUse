package handler

import (
	"net/http"
	"net/url"
	"strings"

	"specuse/backend/internal/account"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/management"
)

func WithAdmin(next http.Handler, auth *account.Service, products *management.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const collection = "/api/v1/admin/products"
		list := r.URL.Path == collection
		id := strings.TrimPrefix(r.URL.Path, collection+"/")
		detail := strings.HasPrefix(r.URL.Path, collection+"/") && id != "" && !strings.Contains(id, "/")
		if !list && !detail {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		actor, ok := identify(w, r, auth)
		if !ok {
			return
		}
		if actor.Profile.Role != "admin" {
			accountError(w, account.ErrForbidden)
			return
		}
		if products == nil {
			accountError(w, account.ErrUnavailable)
			return
		}
		if list {
			if !allowedMethod(w, r, "GET", "POST") {
				return
			}
		} else {
			if !allowedMethod(w, r, "PATCH", "DELETE") {
				return
			}
		}
		if list && r.Method == "GET" {
			values, err := url.ParseQuery(r.URL.RawQuery)
			if err != nil || len(values["category"]) > 1 || len(values["search"]) > 1 {
				accountError(w, account.ErrInput)
				return
			}
			page, err := positiveInt(values, "page", 1)
			if err != nil {
				accountError(w, account.ErrInput)
				return
			}
			size, err := positiveInt(values, "page_size", 20)
			if err != nil {
				accountError(w, account.ErrInput)
				return
			}
			result, err := products.List(r.Context(), actor, catalog.Filter{Category: values.Get("category"), Search: values.Get("search"), Page: page, PageSize: size})
			if err != nil {
				accountError(w, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": result})
			return
		}
		if r.Method == "DELETE" {
			if err := products.Delete(r.Context(), actor, id); err != nil {
				accountError(w, err)
				return
			}
			w.WriteHeader(204)
			return
		}
		var change *management.Change
		if !decodeAccount(w, r, &change) {
			return
		}
		if change == nil {
			accountError(w, account.ErrInput)
			return
		}
		result, err := products.Save(r.Context(), actor, id, *change, r.Method == "POST")
		if err != nil {
			accountError(w, err)
			return
		}
		status := 200
		if r.Method == "POST" {
			status = 201
		}
		writeJSON(w, status, map[string]any{"data": map[string]any{"product": result}})
	})
}
