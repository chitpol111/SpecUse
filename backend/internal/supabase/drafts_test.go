package supabase

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"specuse/backend/internal/account"
	"specuse/backend/internal/draft"
)

func TestDraftRPCUsesOwnerAndServerCredentials(t *testing.T) {
	const actor = "11111111-1111-4111-8111-111111111111"
	const id = "22222222-2222-4222-8222-222222222222"
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Header.Get("apikey") != "server-test" || r.Header.Get("Authorization") != "" {
			t.Error("wrong Data API credentials")
		}
		var args map[string]json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&args)
		var owner string
		_ = json.Unmarshal(args["p_actor"], &owner)
		if owner != actor {
			t.Error("owner not derived from identity")
		}
		if strings.HasSuffix(r.URL.Path, "_list") {
			_, _ = w.Write([]byte(`{"items":[],"page":1,"page_size":20,"total":0}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "_delete") {
			_, _ = w.Write([]byte(`{"deleted":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"22222222-2222-4222-8222-222222222222","name":"Draft","description":"","visibility":"private","created_at":"2026-11-12T00:00:00Z","updated_at":"2026-11-12T00:00:00Z","items":[],"ram_items":[{"ram_model_id":"33333333-3333-4333-8333-333333333333","required_modules":4,"owned_modules":1,"ram_model_summary":{"ram_type":"DDR4"},"active_package_count":0}]}`))
	}))
	defer server.Close()
	c, _ := New(server.URL, "public-test", "server-test", true)
	repo := NewRepository(c)
	ctx := context.Background()
	if _, e := repo.ListDrafts(ctx, actor, draft.Filter{Page: 1, PageSize: 20}); e != nil {
		t.Fatal(e)
	}
	b, e := repo.GetDraft(ctx, actor, id)
	if e != nil || len(b.RAMItems) != 1 || b.RAMItems[0].ActivePackageCount != 0 {
		t.Fatal("RPC row conversion")
	}
	if _, e = repo.CreateDraft(ctx, actor, draft.Change{}); e != nil {
		t.Fatal(e)
	}
	if _, e = repo.PatchDraft(ctx, actor, id, draft.Change{}); e != nil {
		t.Fatal(e)
	}
	if e = repo.DeleteDraft(ctx, actor, id); e != nil || len(paths) != 5 {
		t.Fatal("RPC calls missing")
	}
	raw, _ := json.Marshal(draft.EvaluateAvailability(b))
	if strings.Contains(string(raw), "active_package_count") || !strings.Contains(string(raw), "RAM_PACKAGES_UNAVAILABLE") {
		t.Fatal("internal availability data leaked/lost")
	}
}
func TestDraftRPCRejectsMalformedRecordsAndMapsDatabaseErrors(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `{"id":"22222222-2222-4222-8222-222222222222","visibility":"private","items":[],"ram_items":[{}]}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
		c, _ := New(server.URL, "public", "server", true)
		_, e := NewRepository(c).GetDraft(context.Background(), "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222")
		server.Close()
		if !errors.Is(e, account.ErrUnavailable) {
			t.Fatal("malformed row accepted")
		}
	}
	for _, code := range []string{"22023", "22003"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(400)
			_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": "must not leak"})
		}))
		c, _ := New(server.URL, "public", "server", true)
		e := NewRepository(c).DeleteDraft(context.Background(), "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222")
		server.Close()
		if !errors.Is(e, account.ErrInput) {
			t.Fatal("database input code not mapped")
		}
	}
}
