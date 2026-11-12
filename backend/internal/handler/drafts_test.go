package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"specuse/backend/internal/account"
	"specuse/backend/internal/draft"
	"specuse/backend/internal/supabase"
)

const draftOwner = "11111111-1111-4111-8111-111111111111"
const draftOther = "22222222-2222-4222-8222-222222222222"
const draftID = "33333333-3333-4333-8333-333333333333"

type draftHTTPRepo struct {
	value  draft.Build
	exists bool
	writes int
}

func (r *draftHTTPRepo) GetDraft(_ context.Context, owner, id string) (draft.Build, error) {
	if owner != draftOwner || id != draftID || !r.exists {
		return draft.Build{}, account.ErrNotFound
	}
	return r.value, nil
}
func (r *draftHTTPRepo) ListDrafts(_ context.Context, owner string, f draft.Filter) (draft.Page, error) {
	items := []draft.Summary{}
	if owner == draftOwner && r.exists {
		items = append(items, r.value.Summary)
	}
	return draft.Page{Items: items, Page: f.Page, PageSize: f.PageSize, Total: int64(len(items))}, nil
}
func (r *draftHTTPRepo) apply(c draft.Change) {
	if c.Name != nil {
		r.value.Name = *c.Name
	}
	if c.Description != nil {
		r.value.Description = *c.Description
	}
	if c.Visibility != nil {
		r.value.Visibility = *c.Visibility
	}
	if c.Items != nil {
		r.value.Items = []draft.Item{}
		for _, v := range *c.Items {
			r.value.Items = append(r.value.Items, draft.Item{ItemInput: v, ProductSummary: draft.ProductSummary{Name: "[TEST] Case", Category: "case", IsActive: true}})
		}
	}
	if c.RAMItems != nil {
		r.value.RAMItems = []draft.RAMItem{}
		for _, v := range *c.RAMItems {
			r.value.RAMItems = append(r.value.RAMItems, draft.RAMItem{RAMInput: v, RAMModelSummary: draft.RAMModelSummary{RAMType: "DDR4"}, ActivePackageCount: 1})
		}
	}
	r.writes++
}
func (r *draftHTTPRepo) CreateDraft(_ context.Context, owner string, c draft.Change) (draft.Build, error) {
	if owner != draftOwner {
		return draft.Build{}, account.ErrForbidden
	}
	r.exists = true
	r.value = draft.Build{Summary: draft.Summary{ID: draftID, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	r.apply(c)
	return r.value, nil
}
func (r *draftHTTPRepo) PatchDraft(ctx context.Context, owner, id string, c draft.Change) (draft.Build, error) {
	if _, e := r.GetDraft(ctx, owner, id); e != nil {
		return draft.Build{}, e
	}
	r.apply(c)
	return r.value, nil
}
func (r *draftHTTPRepo) DeleteDraft(ctx context.Context, owner, id string) error {
	if _, e := r.GetDraft(ctx, owner, id); e != nil {
		return e
	}
	r.exists = false
	r.writes++
	return nil
}

type draftProfiles struct{}

func (draftProfiles) GetProfile(_ context.Context, id string) (account.Profile, error) {
	role := "member"
	if id == draftOther {
		role = "admin"
	}
	return account.Profile{ID: id, DisplayName: "Test", Role: role}, nil
}
func (draftProfiles) UpdateProfile(context.Context, string, string) (account.Profile, error) {
	return account.Profile{}, errors.New("unused")
}

type draftProvider struct {
	adminProvider
	tokens map[string]string
}

func (p draftProvider) User(_ context.Context, token string) (json.RawMessage, error) {
	id, ok := p.tokens[token]
	if !ok {
		return nil, account.ErrUnauthorized
	}
	raw, _ := json.Marshal(map[string]string{"id": id, "email": "test@example.test"})
	return raw, nil
}

func TestDraftHTTPContractAuthorizationAndInput(t *testing.T) {
	const issuer = "https://unit.invalid/auth/v1"
	token, otherToken := memberToken(draftOwner, issuer), memberToken(draftOther, issuer)
	auth := account.New(draftProvider{tokens: map[string]string{token: draftOwner, otherToken: draftOther}}, draftProfiles{}, issuer, "authenticated")
	repo := &draftHTTPRepo{}
	h := WithCORS(WithDrafts(http.NotFoundHandler(), auth, draft.New(repo)), []string{"http://localhost:3000", "http://127.0.0.1:3000"}, "GET, POST, PATCH, DELETE")
	call := func(method, path, body, tok string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if tok != "" {
			r.Header.Set("Authorization", "Bearer "+tok)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	const collection = "/api/v1/me/builds"
	for _, method := range []string{"GET", "POST", "PATCH", "DELETE"} {
		path := collection
		if method == "PATCH" || method == "DELETE" {
			path += "/" + draftID
		}
		if w := call(method, path, `{}`, ""); w.Code != 401 {
			t.Fatal("guest not rejected")
		}
	}
	if w := call("GET", collection, "", token+"forged"); w.Code != 401 {
		t.Fatal("invalid token accepted")
	}
	for _, body := range []string{
		`null`, `{}`, `{"name":"Draft","items":[],"ram_items":null}`, `{"name":"Draft","items":[],"ram_items":[],"owner_id":"22222222-2222-4222-8222-222222222222"}`,
		`{"name":"Draft","items":[],"ram_items":[],"role":"admin"}`, `{"name":"Draft","items":[],"ram_items":[],"total_price_thb":0}`,
		`{"name":"Draft","items":[],"ram_items":[],"availability":{"ready_for_validation":true}}`,
		`{"name":"Draft","items":[{"product_id":"44444444-4444-4444-8444-444444444444","quantity":1}],"ram_items":[]}`,
		`{"name":"Draft","items":[{"product_id":"44444444-4444-4444-8444-444444444444","quantity":1,"owned_quantity":0,"product_summary":{}}],"ram_items":[]}`,
		`{"name":"Draft","items":[{"product_id":"44444444-4444-4444-8444-444444444444","quantity":1.5,"owned_quantity":0}],"ram_items":[]}`,
		`{"name":"Draft","items":[],"ram_items":[{"ram_model_id":"55555555-5555-4555-8555-555555555555","required_modules":2}]}`,
		`{"name":"Draft","items":[],"ram_items":[]} {}`,
	} {
		if w := call("POST", collection, body, token); w.Code != 400 {
			t.Fatalf("invalid create HTTP %d", w.Code)
		}
	}
	if repo.writes != 0 {
		t.Fatal("invalid request reached write")
	}
	if w := call("POST", collection, `{"name":"Draft","visibility":"public","items":[],"ram_items":[]}`, token); w.Code != 409 || !strings.Contains(w.Body.String(), "PUBLIC_BUILD_NOT_ENABLED") {
		t.Fatal("public create accepted")
	}
	rawExample, err := os.ReadFile("../../examples/drafts/create-multiple.request.json")
	if err != nil {
		t.Fatal(err)
	}
	valid := string(rawExample)
	w := call("POST", collection, valid, token)
	if w.Code != 201 || w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Body.String(), `"ready_for_validation":true`) {
		t.Fatal("create response incorrect")
	}
	for _, method := range []string{"GET", "PATCH", "DELETE"} {
		if w := call(method, collection+"/"+draftID, `{"name":"Stolen"}`, otherToken); w.Code != 404 {
			t.Fatal("admin accessed another member's build")
		}
	}
	if w := call("GET", collection, "", otherToken); w.Code != 200 || !strings.Contains(w.Body.String(), `"items":[]`) {
		t.Fatal("other owner's list leaked")
	}
	if w := call("PATCH", collection+"/"+draftID, `{"description":"Changed"}`, token); w.Code != 200 || len(repo.value.Items) != 2 || repo.value.Items[0].Quantity != 3 {
		t.Fatal("metadata PATCH dropped hidden items")
	}
	for _, body := range []string{`{}`, `{"items":null}`, `{"name":null}`, `{"description":null}`, `{"id":"other"}`, `{"ram_items":[null]}`} {
		if w := call("PATCH", collection+"/"+draftID, body, token); w.Code != 400 {
			t.Fatal("invalid patch accepted")
		}
	}
	if w := call("PATCH", collection+"/"+draftID, `{"name":"Must not change","visibility":"public"}`, token); w.Code != 409 || repo.value.Name != "[SYNTHETIC EXAMPLE] หลายรายการ" {
		t.Fatal("public patch changed data")
	}
	for _, query := range []string{"?page=0", "?page=1&page=2", "?page_size=", "?owner_id=other"} {
		if w := call("GET", collection+query, "", token); w.Code != 400 {
			t.Fatal("bad list query")
		}
	}
	if w := call("GET", collection+"/bad-uuid", "", token); w.Code != 400 {
		t.Fatal("bad UUID accepted")
	}
	if w := call("POST", collection, valid+strings.Repeat(" ", maxBuildBodyBytes), token); w.Code != 413 {
		t.Fatalf("oversized trailing body HTTP %d", w.Code)
	}
	r := httptest.NewRequest("POST", collection, strings.NewReader(valid))
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "text/plain")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 415 {
		t.Fatal("bad content type")
	}
	for _, origin := range []string{"http://localhost:3000", "http://127.0.0.1:3000"} {
		r := httptest.NewRequest("OPTIONS", collection, nil)
		r.Header.Set("Origin", origin)
		r.Header.Set("Access-Control-Request-Method", "PATCH")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != origin {
			t.Fatal("CORS regression")
		}
	}
	if w := call("DELETE", collection+"/"+draftID, "", token); w.Code != 204 || w.Body.Len() != 0 {
		t.Fatal("delete response")
	}
	if w := call("GET", collection+"/"+draftID, "", token); w.Code != 404 {
		t.Fatal("deleted draft readable")
	}
}

func TestDraftHTTPPostgRESTNotFoundForOtherOwnersAndMissingIDs(t *testing.T) {
	const issuer = "https://unit.invalid/auth/v1"
	const memberB = "44444444-4444-4444-8444-444444444444"
	const missingID = "55555555-5555-4555-8555-555555555555"
	for _, actor := range []struct{ name, id string }{
		{"member_B", memberB}, {"admin_C", draftOther}, {"owner_A", draftOwner},
	} {
		for _, id := range []string{draftID, missingID} {
			if actor.id == draftOwner && id == draftID {
				continue // This test only models hidden or nonexistent records.
			}
			for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodDelete} {
				t.Run(actor.name+"/"+id+"/"+method, func(t *testing.T) {
					calls := 0
					rpcMethod := strings.ToLower(method)
					if method == http.MethodPatch {
						rpcMethod = "get" // PATCH verifies ownership before attempting a write.
					}
					upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls++
						if r.Method != http.MethodPost || r.URL.Path != "/rest/v1/rpc/backend_member_build_"+rpcMethod {
							t.Errorf("unexpected RPC: %s %s", r.Method, r.URL.Path)
						}
						if r.Header.Get("apikey") != "server-test" || r.Header.Get("Authorization") != "" {
							t.Error("wrong privileged RPC credentials")
						}
						var args struct {
							Actor string `json:"p_actor"`
							ID    string `json:"p_id"`
						}
						if err := json.NewDecoder(r.Body).Decode(&args); err != nil || args.Actor != actor.id || args.ID != id {
							t.Errorf("incorrect actor or draft ID: %+v, %v", args, err)
						}
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = w.Write([]byte(`{"code":"P0002","message":"private database detail","details":null,"hint":null}`))
					}))
					defer upstream.Close()
					client, err := supabase.New(upstream.URL, "public-test", "server-test", true)
					if err != nil {
						t.Fatal(err)
					}
					token := memberToken(actor.id, issuer)
					auth := account.New(draftProvider{tokens: map[string]string{token: actor.id}}, draftProfiles{}, issuer, "authenticated")
					h := WithDrafts(http.NotFoundHandler(), auth, draft.New(supabase.NewRepository(client)))
					r := httptest.NewRequest(method, "/api/v1/me/builds/"+id, strings.NewReader(`{"name":"Changed"}`))
					r.Header.Set("Authorization", "Bearer "+token)
					r.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if w.Code != http.StatusNotFound || strings.TrimSpace(w.Body.String()) != `{"error":{"code":"NOT_FOUND","message":"ไม่พบข้อมูล"}}` {
						t.Fatalf("got HTTP %d %s, want the same generic 404 for hidden and missing drafts", w.Code, w.Body.String())
					}
					if calls != 1 {
						t.Fatalf("got %d RPC calls, want 1", calls)
					}
				})
			}
		}
	}
}
