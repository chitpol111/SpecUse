package account

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"specuse/backend/internal/catalog"
)

var (
	ErrInput        = errors.New("invalid account input")
	ErrAuthConfig   = errors.New("authentication configuration does not support immediate signup")
	ErrUnauthorized = errors.New("unauthorized")
	ErrCredentials  = errors.New("invalid credentials")
	ErrRefresh      = errors.New("invalid refresh token")
	ErrConfirm      = errors.New("email not confirmed")
	ErrForbidden    = errors.New("forbidden")
	ErrRejected     = errors.New("auth request rejected")
	ErrReauth       = errors.New("reauthentication required")
	ErrRateLimit    = errors.New("rate limited")
	ErrUnavailable  = errors.New("service unavailable")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
)

type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
}
type AuthResult struct {
	User    json.RawMessage `json:"user"`
	Session *Session        `json:"session"`
}
type Profile struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type Provider interface {
	Register(context.Context, string, string, string) (AuthResult, error)
	Login(context.Context, string, string) (AuthResult, error)
	Refresh(context.Context, string) (AuthResult, error)
	User(context.Context, string) (json.RawMessage, error)
	UpdatePassword(context.Context, string, string) (json.RawMessage, error)
	Logout(context.Context, string) error
}
type Profiles interface {
	GetProfile(context.Context, string) (Profile, error)
	UpdateProfile(context.Context, string, string) (Profile, error)
}
type Identity struct {
	ID, Email, Token string
	User             json.RawMessage
	Profile          Profile
}
type Service struct {
	provider         Provider
	profiles         Profiles
	issuer, audience string
	now              func() time.Time
}

func New(provider Provider, profiles Profiles, issuer, audience string) *Service {
	return &Service{provider: provider, profiles: profiles, issuer: issuer, audience: audience, now: time.Now}
}
func ValidEmail(email string) bool {
	a, err := mail.ParseAddress(email)
	return err == nil && a.Address == email && len(email) <= 254 && strings.Contains(email, "@")
}
func ValidPassword(p string) bool { return utf8.RuneCountInString(p) >= 8 && len(p) <= 1024 }
func DisplayName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if !utf8.ValidString(name) || strings.ContainsRune(name, '\x00') || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 100 {
		return "", ErrInput
	}
	return name, nil
}
func (s *Service) Register(ctx context.Context, email, password, name string) (AuthResult, error) {
	email = strings.TrimSpace(email)
	name, err := DisplayName(name)
	if err != nil || !ValidEmail(email) || !ValidPassword(password) {
		return AuthResult{}, ErrInput
	}
	result, err := s.provider.Register(ctx, email, password, name)
	if err != nil {
		return AuthResult{}, err
	}
	// Supabase must issue the session. Never pretend signup is ready or fall
	// back to password login when confirmation is still enabled upstream.
	if result.Session == nil {
		return AuthResult{}, ErrAuthConfig
	}
	var user struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(result.User, &user) != nil || !catalog.ValidUUID(user.ID) ||
		result.Session.AccessToken == "" || result.Session.RefreshToken == "" || result.Session.TokenType != "bearer" ||
		result.Session.ExpiresIn <= 0 || result.Session.ExpiresAt <= s.now().Unix() {
		return AuthResult{}, ErrUnavailable
	}
	return result, nil
}
func (s *Service) Login(ctx context.Context, email, password string) (AuthResult, error) {
	email = strings.TrimSpace(email)
	if !ValidEmail(email) || password == "" || len(password) > 1024 {
		return AuthResult{}, ErrInput
	}
	return s.provider.Login(ctx, email, password)
}
func (s *Service) Refresh(ctx context.Context, token string) (AuthResult, error) {
	if token == "" || len(token) > 16384 {
		return AuthResult{}, ErrInput
	}
	return s.provider.Refresh(ctx, token)
}

// Claims are prechecked, then the Auth server MUST verify the signature by
// GET /user. Decoding alone never authenticates a request. No JWT secrets here.
func (s *Service) tokenSubject(token string) (string, error) {
	if len(token) > 16384 {
		return "", ErrUnauthorized
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[2] == "" {
		return "", ErrUnauthorized
	}
	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrUnauthorized
	}
	var h struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(header, &h) != nil || h.Alg == "" || strings.EqualFold(h.Alg, "none") {
		return "", ErrUnauthorized
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrUnauthorized
	}
	var c struct {
		Sub string          `json:"sub"`
		Iss string          `json:"iss"`
		Aud json.RawMessage `json:"aud"`
		Exp int64           `json:"exp"`
		Nbf int64           `json:"nbf"`
	}
	if json.Unmarshal(payload, &c) != nil || !catalog.ValidUUID(c.Sub) || c.Iss != s.issuer || c.Exp <= s.now().Unix() || c.Nbf > s.now().Unix() {
		return "", ErrUnauthorized
	}
	var aud string
	var audiences []string
	if json.Unmarshal(c.Aud, &aud) == nil {
		audiences = []string{aud}
	} else if json.Unmarshal(c.Aud, &audiences) != nil {
		return "", ErrUnauthorized
	}
	match := false
	for _, a := range audiences {
		if a == s.audience {
			match = true
		}
	}
	if !match {
		return "", ErrUnauthorized
	}
	return strings.ToLower(c.Sub), nil
}
func (s *Service) Authenticate(ctx context.Context, token string) (Identity, error) {
	id, err := s.tokenSubject(token)
	if err != nil {
		return Identity{}, err
	}
	raw, err := s.provider.User(ctx, token)
	if err != nil {
		return Identity{}, err
	}
	var user struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Anonymous bool   `json:"is_anonymous"`
	}
	if json.Unmarshal(raw, &user) != nil || strings.ToLower(user.ID) != id || user.Anonymous {
		return Identity{}, ErrUnauthorized
	}
	profile, err := s.profiles.GetProfile(ctx, id)
	if err != nil {
		return Identity{}, err
	}
	if profile.ID != id || (profile.Role != "member" && profile.Role != "admin") {
		return Identity{}, ErrUnavailable
	}
	return Identity{ID: id, Email: user.Email, Token: token, User: raw, Profile: profile}, nil
}
func (s *Service) UpdateProfile(ctx context.Context, identity Identity, name string) (Profile, error) {
	name, err := DisplayName(name)
	if err != nil {
		return Profile{}, err
	}
	return s.profiles.UpdateProfile(ctx, identity.ID, name)
}
func (s *Service) Logout(ctx context.Context, identity Identity) error {
	return s.provider.Logout(ctx, identity.Token)
}
func (s *Service) Password(ctx context.Context, identity Identity, current, next string) (AuthResult, error) {
	if current == "" || len(current) > 1024 || !ValidPassword(next) {
		return AuthResult{}, ErrInput
	}
	reauth, err := s.provider.Login(ctx, identity.Email, current)
	if err != nil {
		return AuthResult{}, err
	}
	if reauth.Session == nil {
		return AuthResult{}, ErrUnauthorized
	}
	var user struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(reauth.User, &user) != nil || user.ID != identity.ID {
		return AuthResult{}, ErrUnauthorized
	}
	updated, err := s.provider.UpdatePassword(ctx, reauth.Session.AccessToken, next)
	if err != nil {
		// Clean up the temporary refresh session without logging credentials.
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.provider.Logout(cleanup, reauth.Session.AccessToken)
		return AuthResult{}, err
	}
	reauth.User = updated
	return reauth, nil
}
