// Package supabase uses Supabase Auth and restricted Data API RPCs over HTTPS.
package supabase

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"specuse/backend/internal/account"
)

type Client struct {
	base, publishable, secret string
	http                      *http.Client
}

func New(base, publishable, secret string, local bool) (*Client, error) {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || u.Path != "" || publishable == "" || secret == "" {
		return nil, account.ErrInput
	}
	if u.Scheme != "https" && !(local && u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")) {
		return nil, account.ErrInput
	}
	return &Client{base: strings.TrimSuffix(base, "/"), publishable: publishable, secret: secret, http: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) request(ctx context.Context, method, path, key, token string, body any) (json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, account.ErrInput
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return nil, account.ErrUnavailable
	}
	req.Header.Set("apikey", key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, account.ErrUnavailable
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, account.ErrUnavailable
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return raw, nil
	}
	if res.StatusCode == 429 {
		return nil, account.ErrRateLimit
	}
	if res.StatusCode >= 500 || res.StatusCode >= 300 && res.StatusCode < 400 {
		return nil, account.ErrUnavailable
	}
	var e struct {
		Code      string `json:"code"`
		ErrorCode string `json:"error_code"`
	}
	_ = json.Unmarshal(raw, &e)
	switch e.ErrorCode {
	case "email_not_confirmed":
		return nil, account.ErrConfirm
	case "reauthentication_needed", "reauthentication_not_valid", "insufficient_aal", "mfa_verification_failed":
		return nil, account.ErrReauth
	}
	switch e.Code {
	case "23505", "23503", "23514", "22P02", "PT409":
		return nil, account.ErrConflict
	case "42501":
		return nil, account.ErrForbidden
	case "P0002":
		return nil, account.ErrNotFound
	}
	if res.StatusCode == 401 || res.StatusCode == 403 {
		return nil, account.ErrUnauthorized
	}
	return nil, account.ErrRejected
}
func authResult(raw json.RawMessage) (account.AuthResult, error) {
	var r struct {
		account.Session
		User json.RawMessage `json:"user"`
	}
	if json.Unmarshal(raw, &r) != nil {
		return account.AuthResult{}, account.ErrUnavailable
	}
	if r.AccessToken == "" {
		// Confirm-email signup returns a User directly, not a session envelope.
		var user struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &user) != nil || user.ID == "" {
			return account.AuthResult{}, account.ErrUnavailable
		}
		return account.AuthResult{User: raw}, nil
	}
	if len(r.User) == 0 || r.RefreshToken == "" || r.TokenType != "bearer" {
		return account.AuthResult{}, account.ErrUnavailable
	}
	if r.ExpiresAt == 0 {
		parts := strings.Split(r.AccessToken, ".")
		if len(parts) != 3 {
			return account.AuthResult{}, account.ErrUnavailable
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return account.AuthResult{}, account.ErrUnavailable
		}
		var claims struct {
			Exp int64 `json:"exp"`
		}
		if json.Unmarshal(payload, &claims) != nil {
			return account.AuthResult{}, account.ErrUnavailable
		}
		r.ExpiresAt = claims.Exp
	}
	return account.AuthResult{User: r.User, Session: &r.Session}, nil
}
func (c *Client) Register(ctx context.Context, email, password, name string) (account.AuthResult, error) {
	raw, err := c.request(ctx, "POST", "/auth/v1/signup", c.publishable, "", map[string]any{"email": email, "password": password, "data": map[string]string{"display_name": name}})
	if err != nil {
		return account.AuthResult{}, err
	}
	return authResult(raw)
}
func (c *Client) Login(ctx context.Context, email, password string) (account.AuthResult, error) {
	raw, err := c.request(ctx, "POST", "/auth/v1/token?grant_type=password", c.publishable, "", map[string]string{"email": email, "password": password})
	if err != nil {
		if err == account.ErrRejected || err == account.ErrUnauthorized {
			err = account.ErrCredentials
		}
		return account.AuthResult{}, err
	}
	return authResult(raw)
}
func (c *Client) Refresh(ctx context.Context, token string) (account.AuthResult, error) {
	raw, err := c.request(ctx, "POST", "/auth/v1/token?grant_type=refresh_token", c.publishable, "", map[string]string{"refresh_token": token})
	if err != nil {
		if err == account.ErrRejected || err == account.ErrUnauthorized {
			err = account.ErrRefresh
		}
		return account.AuthResult{}, err
	}
	return authResult(raw)
}
func (c *Client) User(ctx context.Context, token string) (json.RawMessage, error) {
	return c.request(ctx, "GET", "/auth/v1/user", c.publishable, token, nil)
}
func (c *Client) UpdatePassword(ctx context.Context, token, password string) (json.RawMessage, error) {
	return c.request(ctx, "PUT", "/auth/v1/user", c.publishable, token, map[string]string{"password": password})
}
func (c *Client) Logout(ctx context.Context, token string) error {
	_, err := c.request(ctx, "POST", "/auth/v1/logout?scope=local", c.publishable, token, nil)
	return err
}
func (c *Client) RPC(ctx context.Context, name string, args any, target any) error {
	// Modern sb_secret keys belong only in apikey; legacy service JWTs also
	// need Authorization. Never forward a user's token to privileged Data API.
	token := ""
	if strings.Count(c.secret, ".") == 2 {
		token = c.secret
	}
	raw, err := c.request(ctx, "POST", "/rest/v1/rpc/"+name, c.secret, token, args)
	if err != nil {
		if err == account.ErrUnauthorized || err == account.ErrRejected {
			return account.ErrUnavailable
		}
		return err
	}
	if target == nil {
		return nil
	}
	if json.Unmarshal(raw, target) != nil {
		return account.ErrUnavailable
	}
	return nil
}
