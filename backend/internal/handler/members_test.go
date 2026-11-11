package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"specuse/backend/internal/account"
	"specuse/backend/internal/supabase"
)

func memberToken(id, issuer string) string {
	claims, _ := json.Marshal(map[string]any{"sub": id, "iss": issuer, "aud": "authenticated", "exp": time.Now().Add(time.Hour).Unix()})
	return "eyJhbGciOiJIUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(claims) + ".unit-signature"
}
func TestMemberEndpointsOwnershipAndRoleEscalation(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	var token, issuer string
	updates, providerCalls := 0, 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/auth/") {
			providerCalls++
			if r.Header.Get("apikey") != "public-unit" {
				t.Error("server key used for user Auth")
			}
			if r.URL.Path == "/auth/v1/user" {
				if r.Header.Get("Authorization") != "Bearer "+token {
					w.WriteHeader(401)
					return
				}
				_, _ = fmt.Fprintf(w, `{"id":%q,"email":"student@example.test","user_metadata":{"role":"admin"}}`, id)
				return
			}
			if r.URL.Path == "/auth/v1/logout" {
				w.WriteHeader(204)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": token, "refresh_token": "unit-refresh", "token_type": "bearer", "expires_in": 3600, "user": map[string]string{"id": id, "email": "student@example.test"}})
			return
		}
		if r.Header.Get("apikey") != "server-unit" {
			t.Error("Data API not server-only")
		}
		var args map[string]string
		_ = json.NewDecoder(r.Body).Decode(&args)
		if args["p_user_id"] != id {
			t.Error("caller selected another owner's profile")
		}
		name := "Student"
		if strings.HasSuffix(r.URL.Path, "backend_profile_update") {
			updates++
			name = args["p_display_name"]
		}
		_ = json.NewEncoder(w).Encode(account.Profile{ID: id, DisplayName: name, Role: "member", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	}))
	defer backend.Close()
	issuer = backend.URL + "/auth/v1"
	token = memberToken(id, issuer)
	c, _ := supabase.New(backend.URL, "public-unit", "server-unit", true)
	repo := supabase.NewRepository(c)
	h := WithCORS(WithMembers(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, 404, "NOT_FOUND", "ไม่พบ endpoint")
	}), account.New(c, repo, issuer, "authenticated")), []string{"http://localhost:3000"}, "GET, POST, PATCH, DELETE")
	request := func(method, path, body, bearerToken string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if bearerToken != "" {
			r.Header.Set("Authorization", "Bearer "+bearerToken)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, tt := range []struct {
		name, method, path, body, tok string
		want                          int
	}{
		{"register", "POST", "/api/v1/auth/register", `{"email":"student@example.test","password":"example-password","display_name":"Student"}`, "", 201},
		{"register role", "POST", "/api/v1/auth/register", `{"email":"student@example.test","password":"example-password","display_name":"Student","role":"admin"}`, "", 400},
		{"register metadata role", "POST", "/api/v1/auth/register", `{"email":"student@example.test","password":"example-password","display_name":"Student","user_metadata":{"role":"admin"}}`, "", 400},
		{"register app metadata", "POST", "/api/v1/auth/register", `{"email":"student@example.test","password":"example-password","display_name":"Student","app_metadata":{"role":"admin"}}`, "", 400},
		{"login", "POST", "/api/v1/auth/login", `{"email":"student@example.test","password":"example-password"}`, "", 200},
		{"confirm retired", "POST", "/api/v1/auth/confirm", `{}`, "", 410},
		{"refresh", "POST", "/api/v1/auth/refresh", `{"refresh_token":"unit-refresh"}`, "", 200},
		{"profile own", "GET", "/api/v1/me?id=22222222-2222-4222-8222-222222222222", "", token, 200},
		{"profile patch", "PATCH", "/api/v1/me", `{"display_name":"New"}`, token, 200},
		{"patch role", "PATCH", "/api/v1/me", `{"role":"admin"}`, token, 400},
		{"patch owner", "PATCH", "/api/v1/me", `{"display_name":"New","id":"22222222-2222-4222-8222-222222222222"}`, token, 400},
		{"patch email", "PATCH", "/api/v1/me", `{"email":"other@example.test"}`, token, 400},
		{"null patch", "PATCH", "/api/v1/me", `null`, token, 400},
		{"no token", "GET", "/api/v1/me", "", "", 401},
		{"bad signature", "GET", "/api/v1/me", "", token + "forged", 401},
		{"wrong issuer", "GET", "/api/v1/me", "", memberToken(id, "https://other.invalid/auth/v1"), 401},
		{"other owner route", "GET", "/api/v1/me/22222222-2222-4222-8222-222222222222", "", token, 404},
		{"method", "DELETE", "/api/v1/me", "", token, 405},
		{"logout", "POST", "/api/v1/auth/logout", "", token, 204},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := request(tt.method, tt.path, tt.body, tt.tok)
			if w.Code != tt.want {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
			if tt.want != 404 && w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("auth response may be cached")
			}
			if tt.name == "profile own" && (!strings.Contains(w.Body.String(), `"role":"member"`) || strings.Contains(w.Body.String(), `"profile":{"id":"2222`)) {
				t.Fatal("metadata role or owner trusted")
			}
		})
	}
	if updates != 1 || providerCalls == 0 {
		t.Fatal("forbidden fields reached update")
	}
	r := httptest.NewRequest("OPTIONS", "/api/v1/me", nil)
	r.Header.Set("Origin", "http://localhost:3000")
	r.Header.Set("Access-Control-Request-Method", "PATCH")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 || !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "PATCH") {
		t.Fatal("member CORS missing PATCH")
	}
}
