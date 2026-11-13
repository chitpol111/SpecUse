package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"specuse/backend/internal/account"
)

func accountError(w http.ResponseWriter, err error) {
	status, code, message := 500, "INTERNAL_ERROR", "เกิดข้อผิดพลาดภายในเซิร์ฟเวอร์"
	switch {
	case errors.Is(err, account.ErrAuthConfig):
		status, code, message = 503, "AUTH_CONFIGURATION_ERROR", "ระบบสมัครสมาชิกยังตั้งค่าไม่พร้อม กรุณาติดต่อผู้ดูแล"
	case errors.Is(err, account.ErrInput):
		status, code, message = 400, "VALIDATION_ERROR", "ข้อมูลคำขอไม่ถูกต้อง"
	case errors.Is(err, account.ErrUnauthorized):
		status, code, message = 401, "UNAUTHORIZED", "กรุณาเข้าสู่ระบบด้วย token ที่ใช้ได้"
	case errors.Is(err, account.ErrCredentials):
		status, code, message = 401, "INVALID_CREDENTIALS", "อีเมลหรือรหัสผ่านไม่ถูกต้อง"
	case errors.Is(err, account.ErrRefresh):
		status, code, message = 401, "INVALID_REFRESH_TOKEN", "กรุณาเข้าสู่ระบบใหม่"
	case errors.Is(err, account.ErrConfirm):
		status, code, message = 403, "EMAIL_NOT_CONFIRMED", "กรุณายืนยันอีเมลก่อนเข้าสู่ระบบ"
	case errors.Is(err, account.ErrForbidden):
		status, code, message = 403, "FORBIDDEN", "ไม่มีสิทธิ์ใช้งาน"
	case errors.Is(err, account.ErrRejected):
		status, code, message = 400, "AUTH_REQUEST_REJECTED", "ผู้ให้บริการไม่ยอมรับคำขอนี้"
	case errors.Is(err, account.ErrReauth):
		status, code, message = 403, "REAUTHENTICATION_REQUIRED", "ต้องยืนยันตัวตนเพิ่มเติมตามข้อกำหนดของผู้ให้บริการ"
	case errors.Is(err, account.ErrRateLimit):
		status, code, message = 429, "RATE_LIMITED", "กรุณารอก่อนลองใหม่"
	case errors.Is(err, account.ErrUnavailable):
		status, code, message = 503, "SERVICE_UNAVAILABLE", "บริการยังไม่พร้อมใช้งาน"
	case errors.Is(err, account.ErrNotFound):
		status, code, message = 404, "NOT_FOUND", "ไม่พบข้อมูล"
	case errors.Is(err, account.ErrConflict):
		status, code, message = 409, "CONFLICT", "ข้อมูลขัดกับข้อมูลที่มีอยู่"
	}
	writeError(w, status, code, message)
}
func decodeAccount(w http.ResponseWriter, r *http.Request, target any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		writeError(w, 415, "UNSUPPORTED_MEDIA_TYPE", "ต้องใช้ application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	defer r.Body.Close()
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	err = d.Decode(target)
	if err == nil {
		var extra any
		if e := d.Decode(&extra); e != io.EOF {
			err = account.ErrInput
		}
	}
	if err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			writeError(w, 413, "PAYLOAD_TOO_LARGE", "คำขอมีขนาดเกิน 64 KiB")
		} else {
			accountError(w, account.ErrInput)
		}
		return false
	}
	return true
}
func bearer(r *http.Request) (string, error) {
	values := r.Header.Values("Authorization")
	if len(values) != 1 {
		return "", account.ErrUnauthorized
	}
	fields := strings.Fields(values[0])
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return "", account.ErrUnauthorized
	}
	return fields[1], nil
}
func identify(w http.ResponseWriter, r *http.Request, s *account.Service) (account.Identity, bool) {
	token, err := bearer(r)
	if err != nil {
		accountError(w, err)
		return account.Identity{}, false
	}
	if s == nil {
		accountError(w, account.ErrUnavailable)
		return account.Identity{}, false
	}
	id, err := s.Authenticate(r.Context(), token)
	if err != nil {
		accountError(w, err)
		return account.Identity{}, false
	}
	return id, true
}
func allowedMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, m := range methods {
		if r.Method == m {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, 405, "METHOD_NOT_ALLOWED", "ไม่รองรับ HTTP method นี้")
	return false
}
func WithMembers(next http.Handler, s *account.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/api/v1/auth/confirm" {
			w.Header().Set("Cache-Control", "no-store")
			if allowedMethod(w, r, "POST") {
				writeError(w, http.StatusGone, "ENDPOINT_RETIRED", "ไม่ใช้ขั้นตอนยืนยันอีเมลแล้ว กรุณาใช้ Register หรือ Login")
			}
			return
		}
		authRoute := path == "/api/v1/auth/register" || path == "/api/v1/auth/login" || path == "/api/v1/auth/refresh"
		memberRoute := path == "/api/v1/me" || path == "/api/v1/me/password" || path == "/api/v1/auth/logout"
		if !authRoute && !memberRoute {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if authRoute {
			if !allowedMethod(w, r, "POST") {
				return
			}
			if s == nil {
				accountError(w, account.ErrUnavailable)
				return
			}
			var result account.AuthResult
			var err error
			switch path {
			case "/api/v1/auth/register":
				var p *struct {
					Email    string `json:"email"`
					Password string `json:"password"`
					Name     string `json:"display_name"`
				}
				if !decodeAccount(w, r, &p) {
					return
				}
				if p == nil {
					accountError(w, account.ErrInput)
					return
				}
				result, err = s.Register(r.Context(), p.Email, p.Password, p.Name)
			case "/api/v1/auth/login":
				var p *struct {
					Email    string `json:"email"`
					Password string `json:"password"`
				}
				if !decodeAccount(w, r, &p) {
					return
				}
				if p == nil {
					accountError(w, account.ErrInput)
					return
				}
				result, err = s.Login(r.Context(), p.Email, p.Password)
			case "/api/v1/auth/refresh":
				var p *struct {
					Token string `json:"refresh_token"`
				}
				if !decodeAccount(w, r, &p) {
					return
				}
				if p == nil {
					accountError(w, account.ErrInput)
					return
				}
				result, err = s.Refresh(r.Context(), p.Token)
			}
			if err != nil {
				accountError(w, err)
				return
			}
			if path == "/api/v1/auth/register" {
				writeJSON(w, 201, map[string]any{"data": map[string]any{"user": result.User, "session": result.Session, "email_confirmation_required": false}})
			} else {
				writeJSON(w, 200, map[string]any{"data": result})
			}
			return
		}
		switch path {
		case "/api/v1/me":
			if !allowedMethod(w, r, "GET", "PATCH") {
				return
			}
		case "/api/v1/me/password":
			if !allowedMethod(w, r, "PATCH") {
				return
			}
		default:
			if !allowedMethod(w, r, "POST") {
				return
			}
		}
		identity, ok := identify(w, r, s)
		if !ok {
			return
		}
		switch path {
		case "/api/v1/me":
			if r.Method == "GET" {
				writeJSON(w, 200, map[string]any{"data": map[string]any{"user": identity.User, "profile": identity.Profile}})
				return
			}
			var p *struct {
				Name *string `json:"display_name"`
			}
			if !decodeAccount(w, r, &p) {
				return
			}
			if p == nil || p.Name == nil {
				accountError(w, account.ErrInput)
				return
			}
			profile, err := s.UpdateProfile(r.Context(), identity, *p.Name)
			if err != nil {
				accountError(w, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": map[string]any{"profile": profile}})
		case "/api/v1/me/password":
			var p *struct {
				Current string `json:"current_password"`
				Next    string `json:"new_password"`
			}
			if !decodeAccount(w, r, &p) {
				return
			}
			if p == nil {
				accountError(w, account.ErrInput)
				return
			}
			result, err := s.Password(r.Context(), identity, p.Current, p.Next)
			if err != nil {
				accountError(w, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": result})
		case "/api/v1/auth/logout":
			one := make([]byte, 1)
			n, err := r.Body.Read(one)
			if n > 0 || err != nil && err != io.EOF {
				accountError(w, account.ErrInput)
				return
			}
			if err := s.Logout(r.Context(), identity); err != nil {
				accountError(w, err)
				return
			}
			w.WriteHeader(204)
		}
	})
}
