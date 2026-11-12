package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"specuse/backend/internal/account"
	"specuse/backend/internal/draft"
)

func draftError(w http.ResponseWriter, err error) {
	if errors.Is(err, draft.ErrPublicDisabled) {
		writeError(w, 409, "PUBLIC_BUILD_NOT_ENABLED", "ยังไม่เปิดเผยแพร่ Build สาธารณะ")
		return
	}
	accountError(w, err)
}
func decodeDraft(w http.ResponseWriter, r *http.Request, create bool) (draft.Change, bool) {
	var change draft.Change
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		writeError(w, 415, "UNSUPPORTED_MEDIA_TYPE", "ต้องใช้ application/json")
		return change, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBuildBodyBytes)
	defer r.Body.Close()
	d := json.NewDecoder(r.Body)
	var raw map[string]json.RawMessage
	err = d.Decode(&raw)
	if err == nil {
		var extra any
		if e := d.Decode(&extra); e != io.EOF {
			if e == nil {
				err = account.ErrInput
			} else {
				err = e
			}
		}
	}
	if err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			writeError(w, 413, "PAYLOAD_TOO_LARGE", "คำขอมีขนาดเกิน 1 MiB")
		} else {
			accountError(w, account.ErrInput)
		}
		return change, false
	}
	bad := func() (draft.Change, bool) { accountError(w, account.ErrInput); return draft.Change{}, false }
	if len(raw) == 0 {
		return bad()
	}
	for name, value := range raw {
		if strings.TrimSpace(string(value)) == "null" {
			return bad()
		}
		switch name {
		case "name":
			var v string
			if json.Unmarshal(value, &v) != nil {
				return bad()
			}
			change.Name = &v
		case "description":
			var v string
			if json.Unmarshal(value, &v) != nil {
				return bad()
			}
			change.Description = &v
		case "visibility":
			var v string
			if json.Unmarshal(value, &v) != nil {
				return bad()
			}
			change.Visibility = &v
		case "items":
			var rows []buildItemRequest
			decoder := json.NewDecoder(strings.NewReader(string(value)))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&rows) != nil || rows == nil {
				return bad()
			}
			items := make([]draft.ItemInput, 0, len(rows))
			for _, row := range rows {
				if row.Quantity == nil || row.OwnedQuantity == nil {
					return bad()
				}
				items = append(items, draft.ItemInput{ProductID: row.ProductID, Quantity: *row.Quantity, OwnedQuantity: *row.OwnedQuantity})
			}
			change.Items = &items
		case "ram_items":
			var rows []ramItemRequest
			decoder := json.NewDecoder(strings.NewReader(string(value)))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&rows) != nil || rows == nil {
				return bad()
			}
			items := make([]draft.RAMInput, 0, len(rows))
			for _, row := range rows {
				if row.RequiredModules == nil || row.OwnedModules == nil {
					return bad()
				}
				items = append(items, draft.RAMInput{RAMModelID: row.RAMModelID, RequiredModules: *row.RequiredModules, OwnedModules: *row.OwnedModules})
			}
			change.RAMItems = &items
		default:
			return bad()
		}
	}
	if create && (change.Name == nil || change.Items == nil || change.RAMItems == nil) {
		return bad()
	}
	return change, true
}

func WithDrafts(next http.Handler, auth *account.Service, service *draft.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const collection = "/api/v1/me/builds"
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
		if list {
			if !allowedMethod(w, r, "GET", "POST") {
				return
			}
		} else {
			if !allowedMethod(w, r, "GET", "PATCH", "DELETE") {
				return
			}
		}
		if service == nil {
			accountError(w, account.ErrUnavailable)
			return
		}
		if list && r.Method == "GET" {
			q, err := url.ParseQuery(r.URL.RawQuery)
			if err != nil {
				accountError(w, account.ErrInput)
				return
			}
			for key := range q {
				if key != "page" && key != "page_size" {
					accountError(w, account.ErrInput)
					return
				}
			}
			page, err := positiveInt(q, "page", 1)
			if err != nil {
				accountError(w, account.ErrInput)
				return
			}
			size, err := positiveInt(q, "page_size", 20)
			if err != nil {
				accountError(w, account.ErrInput)
				return
			}
			result, err := service.List(r.Context(), actor, draft.Filter{Page: page, PageSize: size})
			if err != nil {
				draftError(w, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": result})
			return
		}
		if detail && r.Method == "GET" {
			result, err := service.Get(r.Context(), actor, id)
			if err != nil {
				draftError(w, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": map[string]any{"build": result}})
			return
		}
		if detail && r.Method == "DELETE" {
			if err := service.Delete(r.Context(), actor, id); err != nil {
				draftError(w, err)
				return
			}
			w.WriteHeader(204)
			return
		}
		change, ok := decodeDraft(w, r, list)
		if !ok {
			return
		}
		var result draft.Build
		var err error
		status := 200
		if list {
			result, err = service.Create(r.Context(), actor, change)
			status = 201
		} else {
			result, err = service.Patch(r.Context(), actor, id, change)
		}
		if err != nil {
			draftError(w, err)
			return
		}
		writeJSON(w, status, map[string]any{"data": map[string]any{"build": result}})
	})
}
