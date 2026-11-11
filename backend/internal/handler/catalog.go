package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"specuse/backend/internal/catalog"
)

type Catalog struct {
	service *catalog.Service
}

func NewCatalog(service *catalog.Service) *Catalog {
	return &Catalog{service: service}
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type productResponse struct {
	Data productData `json:"data"`
}

type productData struct {
	Product catalog.ProductDetails `json:"product"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: errorDetail{Code: code, Message: message}})
}

func serviceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, catalog.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "ข้อมูลคำขอไม่ถูกต้อง")
	case errors.Is(err, catalog.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "ไม่พบสินค้า")
	default:
		log.Printf("catalog request failed: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "เกิดข้อผิดพลาดภายในเซิร์ฟเวอร์")
	}
}

func positiveInt(values url.Values, name string, fallback int) (int, error) {
	raw, exists := values[name]
	if !exists {
		return fallback, nil
	}
	if len(raw) != 1 || raw[0] == "" {
		return 0, catalog.ErrInvalidInput
	}
	for _, c := range raw[0] {
		if c < '0' || c > '9' {
			return 0, catalog.ErrInvalidInput
		}
	}
	value, err := strconv.Atoi(raw[0])
	if err != nil || value < 1 {
		return 0, catalog.ErrInvalidInput
	}
	return value, nil
}

func (h *Catalog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const collection = "/api/v1/products"
	path := r.URL.Path
	isList := path == collection
	id := strings.TrimPrefix(path, collection+"/")
	isDetail := strings.HasPrefix(path, collection+"/") && id != "" && !strings.Contains(id, "/")
	if !isList && !isDetail {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "ไม่พบ endpoint")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ไม่รองรับ HTTP method นี้")
		return
	}
	if isDetail {
		product, err := h.service.Get(r.Context(), id)
		if err != nil {
			serviceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, productResponse{Data: productData{Product: product}})
		return
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(values["category"]) > 1 || len(values["search"]) > 1 {
		serviceError(w, catalog.ErrInvalidInput)
		return
	}
	page, err := positiveInt(values, "page", 1)
	if err != nil {
		serviceError(w, err)
		return
	}
	pageSize, err := positiveInt(values, "page_size", 20)
	if err != nil {
		serviceError(w, err)
		return
	}
	result, err := h.service.List(r.Context(), catalog.Filter{
		Category: values.Get("category"), Search: values.Get("search"), Page: page, PageSize: pageSize,
	})
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Data catalog.Page `json:"data"`
	}{Data: result})
}
