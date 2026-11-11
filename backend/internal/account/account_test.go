package account

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

const userID = "11111111-1111-4111-8111-111111111111"

type providerStub struct {
	token          string
	raw            json.RawMessage
	calls, updates int
	loginErr       error
}

func (p *providerStub) Register(context.Context, string, string, string) (AuthResult, error) {
	return AuthResult{User: p.raw}, nil
}
func (p *providerStub) Confirm(context.Context, string, string) (AuthResult, error) {
	return p.Login(context.Background(), "", "")
}
func (p *providerStub) Login(context.Context, string, string) (AuthResult, error) {
	return AuthResult{User: p.raw, Session: &Session{AccessToken: p.token, RefreshToken: "test-refresh", TokenType: "bearer"}}, p.loginErr
}
func (p *providerStub) Refresh(context.Context, string) (AuthResult, error) {
	return p.Login(context.Background(), "", "")
}
func (p *providerStub) User(_ context.Context, token string) (json.RawMessage, error) {
	p.calls++
	if token != p.token {
		return nil, ErrUnauthorized
	}
	return p.raw, nil
}
func (p *providerStub) UpdatePassword(context.Context, string, string) (json.RawMessage, error) {
	p.updates++
	return p.raw, nil
}
func (p *providerStub) Logout(context.Context, string) error { return nil }

type profileStub struct {
	profile Profile
	seen    []string
}

func (p *profileStub) GetProfile(_ context.Context, id string) (Profile, error) {
	p.seen = append(p.seen, id)
	return p.profile, nil
}
func (p *profileStub) UpdateProfile(_ context.Context, id, name string) (Profile, error) {
	p.seen = append(p.seen, id)
	result := p.profile
	result.DisplayName = name
	return result, nil
}
func jwt(claims map[string]any) string {
	h, _ := json.Marshal(map[string]string{"alg": "HS256"})
	b, _ := json.Marshal(claims)
	return base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(b) + ".test-signature"
}
func fixture() (*Service, *providerStub, *profileStub) {
	p := &providerStub{raw: json.RawMessage(`{"id":"11111111-1111-4111-8111-111111111111","email":"test@example.test","user_metadata":{"role":"admin"}}`)}
	p.token = jwt(map[string]any{"sub": userID, "iss": "https://project.invalid/auth/v1", "aud": "authenticated", "exp": time.Now().Add(time.Hour).Unix()})
	profiles := &profileStub{profile: Profile{ID: userID, Role: "member", DisplayName: "Test"}}
	return New(p, profiles, "https://project.invalid/auth/v1", "authenticated"), p, profiles
}
func TestAuthenticateChecksClaimsAndRequiresProviderVerification(t *testing.T) {
	s, p, profiles := fixture()
	identity, err := s.Authenticate(context.Background(), p.token)
	if err != nil || identity.ID != userID || identity.Profile.Role != "member" || p.calls != 1 {
		t.Fatalf("identity=%+v err=%v", identity, err)
	}
	for _, tt := range []struct {
		name   string
		change func(map[string]any)
	}{
		{"expired", func(c map[string]any) { c["exp"] = time.Now().Unix() - 1 }},
		{"issuer", func(c map[string]any) { c["iss"] = "https://other.invalid/auth/v1" }},
		{"audience", func(c map[string]any) { c["aud"] = "service_role" }},
		{"not yet valid", func(c map[string]any) { c["nbf"] = time.Now().Add(time.Hour).Unix() }},
		{"subject", func(c map[string]any) { c["sub"] = "bad" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := map[string]any{"sub": userID, "iss": s.issuer, "aud": "authenticated", "exp": time.Now().Add(time.Hour).Unix()}
			tt.change(c)
			token := jwt(c)
			if _, err := s.Authenticate(context.Background(), token); !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("accepted %s", tt.name)
			}
		})
	}
	if _, err := s.Authenticate(context.Background(), p.token+"forged"); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("decoded claims bypassed signature verification provider")
	}
	for _, token := range []string{"", "not-jwt", strings.Repeat("x", 16385), "eyJhbGciOiJub25lIn0.e30.signature"} {
		if _, err := s.Authenticate(context.Background(), token); !errors.Is(err, ErrUnauthorized) {
			t.Fatal("malformed accepted")
		}
	}
	if len(profiles.seen) != 1 {
		t.Fatal("unverified identity reached profile repository")
	}
}
func TestProfileOwnershipAndDatabaseRole(t *testing.T) {
	s, p, profiles := fixture()
	identity, err := s.Authenticate(context.Background(), p.token)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.UpdateProfile(context.Background(), identity, " New name ")
	if err != nil || result.DisplayName != "New name" || profiles.seen[1] != userID {
		t.Fatal("owner changed")
	}
	profiles.profile.Role = "admin"
	identity, err = s.Authenticate(context.Background(), p.token)
	if err != nil || identity.Profile.Role != "admin" {
		t.Fatal("role must be read fresh from DB")
	}
	profiles.profile.ID = "22222222-2222-4222-8222-222222222222"
	if _, err := s.Authenticate(context.Background(), p.token); !errors.Is(err, ErrUnavailable) {
		t.Fatal("repository ownership mismatch accepted")
	}
}
func TestPasswordRequiresCurrentPasswordAndReturnsReplacementSession(t *testing.T) {
	s, p, _ := fixture()
	id, err := s.Authenticate(context.Background(), p.token)
	if err != nil {
		t.Fatal(err)
	}
	p.loginErr = ErrCredentials
	if _, err := s.Password(context.Background(), id, "wrong", "new-password"); !errors.Is(err, ErrCredentials) || p.updates != 0 {
		t.Fatal("wrong current password changed password")
	}
	p.loginErr = nil
	result, err := s.Password(context.Background(), id, "old-password", "new-password")
	if err != nil || result.Session == nil || p.updates != 1 {
		t.Fatal("reauth session not returned")
	}
	if _, err := s.Password(context.Background(), id, "old", "short"); !errors.Is(err, ErrInput) {
		t.Fatal("short password accepted")
	}
}
func TestInputPolicy(t *testing.T) {
	s, _, _ := fixture()
	for _, name := range []string{"", strings.Repeat("ก", 101)} {
		if _, err := DisplayName(name); !errors.Is(err, ErrInput) {
			t.Fatal("bad name accepted")
		}
	}
	if _, err := s.Register(context.Background(), "not-email", "password123", "Test"); !errors.Is(err, ErrInput) {
		t.Fatal("bad email")
	}
	if _, err := s.Register(context.Background(), "x@example.test", "short", "Test"); !errors.Is(err, ErrInput) {
		t.Fatal("bad password")
	}
	if _, err := s.Confirm(context.Background(), "x@example.test", "abcdef"); !errors.Is(err, ErrInput) {
		t.Fatal("bad OTP")
	}
	if _, err := s.Refresh(context.Background(), ""); !errors.Is(err, ErrInput) {
		t.Fatal("empty refresh")
	}
}
