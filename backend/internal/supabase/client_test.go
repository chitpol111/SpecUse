package supabase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"specuse/backend/internal/account"
	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
)

func testToken() string {
	claims, _ := json.Marshal(map[string]any{"exp": time.Now().Add(time.Hour).Unix()})
	return "test." + base64.RawURLEncoding.EncodeToString(claims) + ".signature"
}
func TestAuthRESTFlowsAndSafeErrors(t *testing.T) {
	var requests []string
	token := testToken()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		if r.Header.Get("apikey") != "publishable-test" {
			t.Error("wrong Auth key")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/auth/v1/signup" {
			var p map[string]any
			_ = json.NewDecoder(r.Body).Decode(&p)
			if p["data"].(map[string]any)["role"] != nil {
				t.Error("role forwarded")
			}
		}
		if r.URL.Path == "/auth/v1/user" {
			if r.Header.Get("Authorization") != "Bearer "+token {
				w.WriteHeader(401)
				return
			}
			_, _ = w.Write([]byte(`{"id":"11111111-1111-4111-8111-111111111111"}`))
			return
		}
		if r.URL.Path == "/auth/v1/logout" {
			if r.URL.Query().Get("scope") != "local" {
				t.Error("wrong logout scope")
			}
			w.WriteHeader(204)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": token, "refresh_token": "rotated-test-refresh", "token_type": "bearer", "expires_in": 3600, "user": map[string]string{"id": "11111111-1111-4111-8111-111111111111"}})
	}))
	defer server.Close()
	c, err := New(server.URL, "publishable-test", "server-only-test", true)
	if err != nil {
		t.Fatal(err)
	}
	signup, err := c.Register(context.Background(), "x@example.test", "example-password", "Test")
	if err != nil || signup.Session == nil {
		t.Fatal("signup missing immediate provider session")
	}
	for _, flow := range []func() (account.AuthResult, error){func() (account.AuthResult, error) {
		return c.Login(context.Background(), "x@example.test", "example-password")
	}, func() (account.AuthResult, error) { return c.Refresh(context.Background(), "test-refresh") }} {
		r, err := flow()
		if err != nil || r.Session == nil || r.Session.ExpiresAt == 0 || r.Session.RefreshToken != "rotated-test-refresh" {
			t.Fatal("session rotation fields missing")
		}
	}
	if _, err := c.User(context.Background(), token+"forged"); !errors.Is(err, account.ErrUnauthorized) {
		t.Fatal("provider rejection ignored")
	}
	if _, err := c.User(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if _, err := c.UpdatePassword(context.Background(), token, "replacement-password"); err != nil {
		t.Fatal(err)
	}
	if err := c.Logout(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 7 {
		t.Fatalf("unexpected requests %v", requests)
	}
}
func TestDatabaseSnapshotUsesRestrictedRPCAndExistingDomain(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/catalog.local.json")
	if err != nil {
		t.Fatal(err)
	}
	var products []catalog.ProductDetails
	if err := json.Unmarshal(raw, &products); err != nil {
		t.Fatal(err)
	}
	models := []map[string]string{}
	seen := map[string]bool{}
	for _, p := range products {
		if p.Category == "ram" && !seen[p.RAMModelID] {
			models = append(models, map[string]string{"id": p.RAMModelID, "ram_type": p.Specs.RAMType})
			seen[p.RAMModelID] = true
		}
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/rest/v1/rpc/backend_catalog_snapshot" || r.Header.Get("apikey") != "server-only-test" || r.Header.Get("Authorization") != "" {
			t.Error("credential/RPC boundary violated")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"products": products, "ram_models": models})
	}))
	defer server.Close()
	c, _ := New(server.URL, "public-test", "server-only-test", true)
	repo := NewRepository(c)
	rows, total, err := repo.ListActive(context.Background(), catalog.Filter{Page: 1, PageSize: 20})
	if err != nil || total != 7 || len(rows) != 7 {
		t.Fatal("catalog snapshot changed")
	}
	result, err := build.NewService(repo).Validate(context.Background(), []build.Item{{ProductID: "abcdefab-0000-0000-0000-000000000001", Quantity: 1, OwnedQuantity: 1}, {ProductID: "abcdefab-0000-0000-0000-000000000004", Quantity: 1}}, []build.RAMItem{{RAMModelID: "10000000-0000-0000-0000-000000000001", RequiredModules: 3}})
	if err != nil || result.TotalPriceTHB != 11500 || !result.Compatible || calls != 2 {
		t.Fatalf("snapshot regression %+v %v calls=%d", result, err, calls)
	}
}
func TestConfigErrorsProviderFailuresAndNoRedirectCredentialLeak(t *testing.T) {
	if _, err := New("http://project.invalid", "a", "b", false); err == nil {
		t.Fatal("insecure production URL accepted")
	}

	if _, err := New("https://user:password@project.invalid", "a", "b", false); err == nil {
		t.Fatal("credentials in URL accepted")
	}
	sinkCalls := 0
	sink := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { sinkCalls++ }))
	defer sink.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, sink.URL, 302) }))
	defer server.Close()
	c, _ := New(server.URL, "public-test", "server-test", true)
	if _, err := c.User(context.Background(), "token"); !errors.Is(err, account.ErrUnavailable) || sinkCalls != 0 {
		t.Fatal("redirect followed with secrets")
	}
	for _, tt := range []struct {
		status int
		body   string
		want   error
	}{
		{429, `{}`, account.ErrRateLimit}, {500, `{"secret":"not-returned"}`, account.ErrUnavailable}, {400, `{"error_code":"email_not_confirmed"}`, account.ErrConfirm}, {400, `{"error_code":"reauthentication_needed"}`, account.ErrReauth},
		{409, `{"code":"PT409"}`, account.ErrConflict}, {404, `{"code":"P0002"}`, account.ErrNotFound},
	} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tt.status)
			_, _ = w.Write([]byte(tt.body))
		}))
		client, _ := New(s.URL, "pub", "server", true)
		_, err := client.Login(context.Background(), "x@example.test", "password")
		s.Close()
		if !errors.Is(err, tt.want) || strings.Contains(err.Error(), "not-returned") {
			t.Fatalf("unsafe error %v", err)
		}
	}
}

func TestPostgRESTNotFoundPrecedesServerErrorFallback(t *testing.T) {
	for _, tt := range []struct {
		name   string
		path   string
		status int
		body   string
		want   error
	}{
		{"RPC no_data_found with HTTP 500", "/rest/v1/rpc/backend_member_build_get", 500, `{"code":"P0002","message":"private database detail","details":null,"hint":null}`, account.ErrNotFound},
		{"PATCH RPC no_data_found", "/rest/v1/rpc/backend_member_build_patch", 500, `{"code":"P0002"}`, account.ErrNotFound},
		{"DELETE RPC no_data_found", "/rest/v1/rpc/backend_member_build_delete", 500, `{"code":"P0002"}`, account.ErrNotFound},
		{"RPC no_data_found with HTTP 404", "/rest/v1/rpc/backend_member_build_get", 404, `{"code":"P0002"}`, account.ErrNotFound},
		{"unknown SQLSTATE", "/rest/v1/rpc/backend_member_build_get", 500, `{"code":"XX000","message":"private database detail"}`, account.ErrUnavailable},
		{"unclassified server error", "/rest/v1/rpc/backend_member_build_get", 500, `{"message":"not found"}`, account.ErrUnavailable},
		{"unavailable upstream", "/rest/v1/rpc/backend_member_build_get", 503, `{}`, account.ErrUnavailable},
		{"gateway HTML", "/rest/v1/rpc/backend_member_build_get", 502, `<html>Bad gateway</html>`, account.ErrUnavailable},
		{"malformed JSON", "/rest/v1/rpc/backend_member_build_get", 500, `{"code":"P0002"`, account.ErrUnavailable},
		{"invalid error shape", "/rest/v1/rpc/backend_member_build_get", 500, `{"code":"P0002","error_code":123}`, account.ErrUnavailable},
		{"redirect cannot become not found", "/rest/v1/rpc/backend_member_build_get", 302, `{"code":"P0002"}`, account.ErrUnavailable},
		{"rate limit stays rate limit", "/rest/v1/rpc/backend_member_build_get", 429, `{"code":"P0002"}`, account.ErrRateLimit},
		{"Auth server errors are unchanged", "/auth/v1/user", 500, `{"code":"P0002"}`, account.ErrUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			client, err := New(server.URL, "public-test", "server-test", true)
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.request(context.Background(), http.MethodPost, tt.path, "server-test", "", nil)
			if !errors.Is(err, tt.want) {
				t.Fatalf("HTTP %d: got %v, want %v", tt.status, err, tt.want)
			}
		})
	}
}

func TestProductSaveUsesOriginalDatabaseSnapshot(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	const original = `{"id":"11111111-1111-4111-8111-111111111111","name":"GPU","category":"gpu","brand":"Demo","price_thb":100,"is_active":true,"specs":null}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/v1/rpc/backend_catalog_snapshot":
			_, _ = w.Write([]byte(`{"products":[` + original + `],"ram_models":[]}`))
		case "/rest/v1/rpc/backend_product_save_checked":
			var body struct {
				Expected json.RawMessage `json:"p_expected"`
				Create   bool            `json:"p_create"`
			}
			if json.NewDecoder(r.Body).Decode(&body) != nil || body.Create || string(body.Expected) != original {
				t.Error("original database product was not forwarded")
			}
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"code":"PT409"}`))
		default:
			t.Errorf("unexpected RPC: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := New(server.URL, "public-test", "server-test", true)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(client)
	snapshot, originals, err := repo.LoadSnapshotForWrite(context.Background())
	if err != nil || len(snapshot.Products) != 1 || string(originals[id]) != original {
		t.Fatalf("snapshot or original product lost: %v", err)
	}
	product := catalog.ProductDetails{Product: snapshot.Products[0].Product}
	if _, err := repo.SaveProduct(context.Background(), id, product, false, originals[id]); !errors.Is(err, account.ErrConflict) {
		t.Fatalf("expected stale product conflict, got %v", err)
	}
}
