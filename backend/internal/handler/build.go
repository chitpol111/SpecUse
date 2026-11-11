package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"

	"specuse/backend/internal/build"
)

const maxBuildBodyBytes = 1 << 20

type BuildValidation struct{ service *build.Service }

func NewBuildValidation(service *build.Service) *BuildValidation {
	return &BuildValidation{service: service}
}

type buildItemRequest struct {
	ProductID     string `json:"product_id"`
	Quantity      *int64 `json:"quantity"`
	OwnedQuantity *int64 `json:"owned_quantity"`
}

type ramItemRequest struct {
	RAMModelID      string `json:"ram_model_id"`
	RequiredModules *int64 `json:"required_modules"`
	OwnedModules    *int64 `json:"owned_modules"`
}

type buildRequest struct {
	Items    *[]buildItemRequest `json:"items"`
	RAMItems *[]ramItemRequest   `json:"ram_items"`
}

type buildIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type pricedItem struct {
	ProductID        string `json:"product_id"`
	Quantity         int64  `json:"quantity"`
	OwnedQuantity    int64  `json:"owned_quantity"`
	PurchaseQuantity int64  `json:"purchase_quantity"`
	UnitPriceTHB     int64  `json:"unit_price_thb"`
	PurchasePriceTHB int64  `json:"purchase_price_thb"`
}

type ramPackage struct {
	ProductID         string `json:"product_id"`
	Quantity          int64  `json:"quantity"`
	ModulesPerPackage int64  `json:"modules_per_package"`
	UnitPriceTHB      int64  `json:"unit_price_thb"`
}

type ramSelection struct {
	RAMModelID      string       `json:"ram_model_id"`
	RequiredModules int64        `json:"required_modules"`
	OwnedModules    int64        `json:"owned_modules"`
	PurchaseModules int64        `json:"purchase_modules"`
	SuppliedModules int64        `json:"supplied_modules"`
	ExcessModules   int64        `json:"excess_modules"`
	Packages        []ramPackage `json:"packages"`
	TotalPriceTHB   int64        `json:"total_price_thb"`
}

type buildData struct {
	Compatible    bool           `json:"compatible"`
	Issues        []buildIssue   `json:"issues"`
	Items         []pricedItem   `json:"items"`
	RAMSelections []ramSelection `json:"ram_selections"`
	TotalPriceTHB int64          `json:"total_price_thb"`
}

type buildResponse struct {
	Data buildData `json:"data"`
}

func publicBuildResult(result build.Result) buildResponse {
	data := buildData{Compatible: result.Compatible, TotalPriceTHB: result.TotalPriceTHB,
		Issues: []buildIssue{}, Items: []pricedItem{}, RAMSelections: []ramSelection{}}
	for _, issue := range result.Issues {
		message := "พบปัญหาความเข้ากันได้ของอุปกรณ์"
		switch issue.Code {
		case "CPU_SOCKET_MISMATCH":
			message = "CPU และเมนบอร์ดใช้ซ็อกเก็ตต่างกัน"
		case "RAM_TYPE_MISMATCH":
			message = "RAM และเมนบอร์ดใช้ RAM type ต่างกัน"
		case "MISSING_CPU_SOCKET":
			message = "ข้อมูล CPU socket ของสินค้าไม่ครบ"
		case "MISSING_RAM_TYPE":
			message = "ข้อมูล RAM type ไม่ครบ"
		}
		data.Issues = append(data.Issues, buildIssue{Code: issue.Code, Message: message})
	}
	for _, item := range result.Items {
		data.Items = append(data.Items, pricedItem{ProductID: item.ProductID, Quantity: item.Quantity,
			OwnedQuantity: item.OwnedQuantity, PurchaseQuantity: item.PurchaseQuantity,
			UnitPriceTHB: item.UnitPriceTHB, PurchasePriceTHB: item.PurchasePriceTHB})
	}
	for _, selection := range result.RAMSelections {
		entry := ramSelection{RAMModelID: selection.RAMModelID, RequiredModules: selection.RequiredModules,
			OwnedModules: selection.OwnedModules, PurchaseModules: selection.PurchaseModules,
			SuppliedModules: selection.SuppliedModules, ExcessModules: selection.ExcessModules,
			TotalPriceTHB: selection.TotalPriceTHB, Packages: []ramPackage{}}
		for _, pack := range selection.Packages {
			entry.Packages = append(entry.Packages, ramPackage{ProductID: pack.ProductID, Quantity: pack.Quantity,
				ModulesPerPackage: pack.ModulesPerPackage, UnitPriceTHB: pack.UnitPriceTHB})
		}
		data.RAMSelections = append(data.RAMSelections, entry)
	}
	return buildResponse{Data: data}
}

func buildError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "คำขอมีขนาดเกิน 1 MiB")
	case errors.Is(err, build.ErrInvalidInput), errors.Is(err, build.ErrOverflow):
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "ข้อมูลรายการอุปกรณ์หรือจำนวนไม่ถูกต้อง")
	case errors.Is(err, build.ErrUnavailable):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "ไม่พบสินค้าหรือแพ็ก RAM ที่พร้อมใช้งาน")
	default:
		log.Printf("build validation failed: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "เกิดข้อผิดพลาดภายในเซิร์ฟเวอร์")
	}
}

func decodeBuildError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		buildError(w, err)
		return
	}
	buildError(w, build.ErrInvalidInput)
}

func (h *BuildValidation) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "ไม่รองรับ HTTP method นี้")
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "กรุณาส่ง Content-Type: application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBuildBodyBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var payload *buildRequest
	if err := decoder.Decode(&payload); err != nil {
		decodeBuildError(w, err)
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = build.ErrInvalidInput
		}
		decodeBuildError(w, err)
		return
	}
	if payload == nil || payload.Items == nil || payload.RAMItems == nil {
		buildError(w, build.ErrInvalidInput)
		return
	}
	items := make([]build.Item, 0, len(*payload.Items))
	for _, item := range *payload.Items {
		if item.Quantity == nil || item.OwnedQuantity == nil {
			buildError(w, build.ErrInvalidInput)
			return
		}
		items = append(items, build.Item{ProductID: item.ProductID, Quantity: *item.Quantity, OwnedQuantity: *item.OwnedQuantity})
	}
	ramItems := make([]build.RAMItem, 0, len(*payload.RAMItems))
	for _, item := range *payload.RAMItems {
		if item.RequiredModules == nil || item.OwnedModules == nil {
			buildError(w, build.ErrInvalidInput)
			return
		}
		ramItems = append(ramItems, build.RAMItem{RAMModelID: item.RAMModelID, RequiredModules: *item.RequiredModules, OwnedModules: *item.OwnedModules})
	}
	result, err := h.service.Validate(r.Context(), items, ramItems)
	if err != nil {
		buildError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, publicBuildResult(result))
}
