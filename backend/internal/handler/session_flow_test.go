package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"specuse/backend/internal/account"
	"specuse/backend/internal/supabase"
)

// This models provider session state only. It is not a hosted Supabase test or
// proof of Supabase's cryptographic token verification.
func TestImmediateRegistrationSessionLifecycle(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	const email = "student@example.test"
	password := "EXAMPLE-ONLY-initial"
	var mutex sync.Mutex
	var issuer string
	sequence := 0
	access := map[string]string{}
	refresh := map[string]bool{}
	user := map[string]any{"id": id, "email": email, "user_metadata": map[string]string{"role": "admin"}}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		var body map[string]any
		if r.Method != "GET" {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		fail := func() { w.WriteHeader(401); _, _ = w.Write([]byte(`{"error_code":"invalid_credentials"}`)) }
		issue := func() {
			sequence++
			claims, _ := json.Marshal(map[string]any{"sub": id, "iss": issuer, "aud": "authenticated", "exp": time.Now().Add(time.Hour).Unix(), "jti": sequence})
			token := "eyJhbGciOiJIUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(claims) + ".test-signature"
			ref := fmt.Sprintf("TEST-REFRESH-%d", sequence)
			access[token], refresh[ref] = ref, true
			_ = json.NewEncoder(w).Encode(map[string]any{"user": user, "access_token": token, "refresh_token": ref, "token_type": "bearer", "expires_in": 3600})
		}
		if strings.HasPrefix(r.URL.Path, "/auth/") {
			if r.Header.Get("apikey") != "public-test" {
				t.Error("privileged key sent to Auth")
			}
			switch r.URL.Path {
			case "/auth/v1/signup":
				if body["email"] != email || body["password"] != password {
					fail()
					return
				}
				metadata, _ := body["data"].(map[string]any)
				if metadata["role"] != nil || len(metadata) != 1 {
					t.Error("signup metadata was not whitelisted")
				}
				issue()
			case "/auth/v1/token":
				if r.URL.Query().Get("grant_type") == "refresh_token" {
					ref, _ := body["refresh_token"].(string)
					if !refresh[ref] {
						fail()
						return
					}
					delete(refresh, ref)
				} else if body["email"] != email || body["password"] != password {
					fail()
					return
				}
				issue()
			case "/auth/v1/user":
				token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
				if access[token] == "" {
					fail()
					return
				}
				if r.Method == "PUT" {
					password, _ = body["password"].(string)
				}
				_ = json.NewEncoder(w).Encode(user)
			case "/auth/v1/logout":
				if r.URL.Query().Get("scope") != "local" {
					t.Error("logout scope changed")
				}
				token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
				delete(refresh, access[token]) // Access JWT is intentionally not revoked here.
				w.WriteHeader(204)
			default:
				t.Error("unexpected Auth path")
				w.WriteHeader(404)
			}
			return
		}
		if r.Header.Get("apikey") != "server-test" || body["p_user_id"] != id {
			t.Error("database key/owner boundary violated")
		}
		_ = json.NewEncoder(w).Encode(account.Profile{ID: id, DisplayName: "Student", Role: "member", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	}))
	defer provider.Close()
	issuer = provider.URL + "/auth/v1"
	c, _ := supabase.New(provider.URL, "public-test", "server-test", true)
	h := WithMembers(http.NotFoundHandler(), account.New(c, supabase.NewRepository(c), issuer, "authenticated"))
	call := func(method, path string, body any, token string, want int) json.RawMessage {
		t.Helper()
		var encoded []byte
		if body != nil {
			encoded, _ = json.Marshal(body)
		}
		r := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(string(encoded)))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: got %d want %d", method, path, w.Code, want)
		}
		return w.Body.Bytes()
	}
	readSession := func(raw json.RawMessage) account.Session {
		t.Helper()
		var response struct {
			Data struct {
				Session      *account.Session `json:"session"`
				Confirmation *bool            `json:"email_confirmation_required"`
			} `json:"data"`
		}
		if json.Unmarshal(raw, &response) != nil || response.Data.Session == nil {
			t.Fatal("session missing")
		}
		if response.Data.Confirmation != nil && *response.Data.Confirmation {
			t.Fatal("Frontend told to wait for OTP")
		}
		return *response.Data.Session
	}
	call("GET", "/me", nil, "", 401)
	registration := call("POST", "/auth/register", map[string]string{"email": email, "password": "EXAMPLE-ONLY-initial", "display_name": "Student"}, "", 201)
	if !strings.Contains(string(registration), `"email_confirmation_required":false`) {
		t.Fatal("registration confirmation flag missing or true")
	}
	registered := readSession(registration)
	profile := call("GET", "/me", nil, registered.AccessToken, 200)
	var me struct {
		Data struct {
			Profile account.Profile `json:"profile"`
		} `json:"data"`
	}
	if json.Unmarshal(profile, &me) != nil || me.Data.Profile.ID != id || me.Data.Profile.Role != "member" {
		t.Fatal("register session cannot access own member profile")
	}
	call("PATCH", "/me", map[string]string{"role": "admin"}, registered.AccessToken, 400)
	logged := readSession(call("POST", "/auth/login", map[string]string{"email": email, "password": "EXAMPLE-ONLY-initial"}, "", 200))
	rotated := readSession(call("POST", "/auth/refresh", map[string]string{"refresh_token": logged.RefreshToken}, "", 200))
	if rotated.RefreshToken == logged.RefreshToken {
		t.Fatal("refresh pair not replaced")
	}
	call("POST", "/auth/refresh", map[string]string{"refresh_token": logged.RefreshToken}, "", 401)
	call("PATCH", "/me/password", map[string]string{"current_password": "wrong", "new_password": "EXAMPLE-ONLY-next"}, rotated.AccessToken, 401)
	changed := readSession(call("PATCH", "/me/password", map[string]string{"current_password": "EXAMPLE-ONLY-initial", "new_password": "EXAMPLE-ONLY-next"}, rotated.AccessToken, 200))
	call("GET", "/me", nil, changed.AccessToken, 200)
	call("POST", "/auth/login", map[string]string{"email": email, "password": "EXAMPLE-ONLY-initial"}, "", 401)
	call("POST", "/auth/logout", nil, changed.AccessToken, 204)
	call("POST", "/auth/refresh", map[string]string{"refresh_token": changed.RefreshToken}, "", 401)
	call("GET", "/me", nil, changed.AccessToken, 200) // Do not claim immediate access-JWT revocation.
}

func TestConfirmationConfigurationFailureAndRetiredEndpoint(t *testing.T) {
	calls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/auth/v1/signup" {
			t.Error("confirmation configuration bypassed via another Auth endpoint")
		}
		_, _ = w.Write([]byte(`{"id":"11111111-1111-4111-8111-111111111111","email":"student@example.test"}`))
	}))
	defer provider.Close()
	c, _ := supabase.New(provider.URL, "public-test", "server-test", true)
	h := WithMembers(http.NotFoundHandler(), account.New(c, supabase.NewRepository(c), provider.URL+"/auth/v1", "authenticated"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{"email":"student@example.test","password":"EXAMPLE-ONLY-password","display_name":"Student"}`))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)
	if w.Code != 503 || !strings.Contains(w.Body.String(), `"code":"AUTH_CONFIGURATION_ERROR"`) || strings.Contains(w.Body.String(), `"data"`) {
		t.Fatal("no-session registration reported ready")
	}
	for _, service := range []*account.Service{nil, account.New(c, supabase.NewRepository(c), provider.URL+"/auth/v1", "authenticated")} {
		for _, method := range []string{"POST", "GET"} {
			w = httptest.NewRecorder()
			WithMembers(http.NotFoundHandler(), service).ServeHTTP(w, httptest.NewRequest(method, "/api/v1/auth/confirm", strings.NewReader(`{"token":"123456"}`)))
			want := 410
			if method == "GET" {
				want = 405
			}
			if w.Code != want || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("retired confirm endpoint behavior changed")
			}
		}
	}
	if calls != 1 {
		t.Fatal("retired endpoint contacted provider or signup auto-login fallback occurred")
	}
}
