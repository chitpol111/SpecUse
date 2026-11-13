package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"specuse/backend/internal/account"
	"specuse/backend/internal/build"
	"specuse/backend/internal/catalog"
	"specuse/backend/internal/management"
)

type adminProvider struct{ user json.RawMessage }

func (p adminProvider) Register(context.Context, string, string, string) (account.AuthResult, error) {
	return account.AuthResult{}, nil
}
func (p adminProvider) Login(context.Context, string, string) (account.AuthResult, error) {
	return account.AuthResult{}, nil
}
func (p adminProvider) Refresh(context.Context, string) (account.AuthResult, error) {
	return account.AuthResult{}, nil
}
func (p adminProvider) User(context.Context, string) (json.RawMessage, error) { return p.user, nil }
func (p adminProvider) UpdatePassword(context.Context, string, string) (json.RawMessage, error) {
	return p.user, nil
}
func (p adminProvider) Logout(context.Context, string) error { return nil }

type adminRepo struct {
	role   string
	writes int
}

func (r *adminRepo) GetProfile(context.Context, string) (account.Profile, error) {
	return account.Profile{ID: "11111111-1111-4111-8111-111111111111", Role: r.role}, nil
}
func (r *adminRepo) UpdateProfile(context.Context, string, string) (account.Profile, error) {
	return account.Profile{}, nil
}
func (r *adminRepo) LoadSnapshot(context.Context) (build.Snapshot, error) {
	return build.Snapshot{Products: []build.Component{{Product: catalog.Product{ID: "22222222-2222-4222-8222-222222222222", Name: "Demo", Category: "case", Brand: "Demo", PriceTHB: 100, IsActive: true}}}}, nil
}
func (r *adminRepo) LoadSnapshotForWrite(ctx context.Context) (build.Snapshot, map[string]json.RawMessage, error) {
	snapshot, err := r.LoadSnapshot(ctx)
	expected := make(map[string]json.RawMessage)
	for _, c := range snapshot.Products {
		expected[c.Product.ID], _ = json.Marshal(c.Product)
	}
	return snapshot, expected, err
}
func (r *adminRepo) SaveProduct(_ context.Context, actor string, p catalog.ProductDetails, create bool, expected json.RawMessage) (catalog.ProductDetails, error) {
	r.writes++
	return p, nil
}
func (r *adminRepo) DeactivateProduct(context.Context, string, string) error { r.writes++; return nil }
func TestAdminHTTPAuthorizationAndCRUD(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	const product = "22222222-2222-4222-8222-222222222222"
	issuer := "https://unit.invalid/auth/v1"
	token := memberToken(id, issuer)
	repo := &adminRepo{role: "member"}
	auth := account.New(adminProvider{json.RawMessage(`{"id":"11111111-1111-4111-8111-111111111111","user_metadata":{"role":"admin"}}`)}, repo, issuer, "authenticated")
	h := WithAdmin(http.NotFoundHandler(), auth, management.New(repo))
	request := func(method, path, body, tok string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if tok != "" {
			r.Header.Set("Authorization", "Bearer "+tok)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, method := range []string{"GET", "POST", "PATCH", "DELETE"} {
		path := "/api/v1/admin/products"
		if method == "PATCH" || method == "DELETE" {
			path += "/" + product
		}
		if w := request(method, path, `{}`, ""); w.Code != 401 {
			t.Fatal("guest not 401")
		}
		if w := request(method, path, `{}`, token); w.Code != 403 {
			t.Fatal("member/metadata admin not 403")
		}
	}
	if repo.writes != 0 {
		t.Fatal("member mutation reached DB")
	}
	repo.role = "admin" // authoritative DB change takes effect without a new JWT
	for _, tt := range []struct {
		method, path, body string
		want               int
	}{
		{"GET", "/api/v1/admin/products", "", 200},
		{"POST", "/api/v1/admin/products", `{"name":"Case","category":"case","brand":"Demo","price_thb":200,"is_active":true}`, 201},
		{"PATCH", "/api/v1/admin/products/" + product, `{"price_thb":300}`, 200},
		{"DELETE", "/api/v1/admin/products/" + product, "", 204},
		{"POST", "/api/v1/admin/products", `{"id":"22222222-2222-4222-8222-222222222222","name":"X"}`, 400},
	} {
		w := request(tt.method, tt.path, tt.body, token)
		if w.Code != tt.want {
			t.Fatalf("%s %d %s", tt.method, w.Code, w.Body.String())
		}
	}
	if repo.writes != 3 {
		t.Fatal("CRUD count")
	}
}

type signupAdminProvider struct {
	adminProvider
	session account.Session
}

func (p signupAdminProvider) Register(context.Context, string, string, string) (account.AuthResult, error) {
	return account.AuthResult{User: p.user, Session: &p.session}, nil
}
func TestRegisteredMemberSessionCannotAccessAdmin(t *testing.T) {
	const issuer = "https://unit.invalid/auth/v1"
	const id = "11111111-1111-4111-8111-111111111111"
	token := memberToken(id, issuer)
	provider := signupAdminProvider{adminProvider: adminProvider{json.RawMessage(`{"id":"11111111-1111-4111-8111-111111111111","email":"student@example.test","user_metadata":{"role":"admin"}}`)}, session: account.Session{AccessToken: token, RefreshToken: "TEST-REFRESH", TokenType: "bearer", ExpiresIn: 3600, ExpiresAt: time.Now().Add(time.Hour).Unix()}}
	repo := &adminRepo{role: "member"}
	auth := account.New(provider, repo, issuer, "authenticated")
	h := WithAdmin(WithMembers(http.NotFoundHandler(), auth), auth, management.New(repo))
	register := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{"email":"student@example.test","password":"EXAMPLE-ONLY-password","display_name":"Student"}`))
	register.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, register)
	if w.Code != 201 {
		t.Fatal("registration failed")
	}
	var result struct {
		Data account.AuthResult `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &result) != nil || result.Data.Session == nil {
		t.Fatal("signup did not provide a session")
	}
	tok := result.Data.Session.AccessToken
	for _, tt := range []struct {
		method, path string
		want         int
	}{
		{"GET", "/api/v1/me", 200},
		{"GET", "/api/v1/admin/products", 403},
		{"POST", "/api/v1/admin/products", 403},
		{"PATCH", "/api/v1/admin/products/22222222-2222-4222-8222-222222222222", 403},
		{"DELETE", "/api/v1/admin/products/22222222-2222-4222-8222-222222222222", 403},
	} {
		r := httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{}`))
		r.Header.Set("Authorization", "Bearer "+tok)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tt.want {
			t.Fatalf("%s %s: %d", tt.method, tt.path, w.Code)
		}
	}
	if repo.writes != 0 {
		t.Fatal("registered member reached admin writes")
	}
}
