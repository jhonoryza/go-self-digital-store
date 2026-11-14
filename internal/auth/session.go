package auth

// Sesi login: token acak di cookie, DB menyimpan sha256(token); CSRF stateless
// diturunkan dari token sesi (sha256(token+":csrf")) — bekerja lintas origin
// karena dikirim sebagai header X-CSRF-Token dari props Inertia, bukan cookie
// ganda. Cookie SameSite dikonfigurasi lewat config (None untuk split-origin).

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

const (
	sessionTTL   = 30 * 24 * time.Hour
	sessionCookie = "store_session"
)

// Token acak 32 byte -> 64 hex.
func Token() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// CSRFToken token CSRF milik sesi.
func CSRFToken(sessionRawToken string) string {
	return hashToken(sessionRawToken + ":csrf")
}

// StartSession buat sesi baru untuk user, set cookie, kembalikan token mentah.
func StartSession(db *sql.DB, w http.ResponseWriter, userID int64, cfg CookieConfig) (string, error) {
	raw := Token()
	if _, err := db.Exec(`INSERT INTO sessions (token, user_id, csrf_token, expires_at) VALUES (?,?,?,?)`,
		hashToken(raw), userID, CSRFToken(raw), time.Now().Add(sessionTTL).UTC().Format(time.DateTime)); err != nil {
		return "", err
	}
	db.Exec(`DELETE FROM sessions WHERE expires_at < datetime('now')`) // rapikan kedaluwarsa
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    raw,
		Path:     "/",
		Expires:  time.Now().Add(sessionTTL),
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: sameSite(cfg.SameSite),
	})
	return raw, nil
}

// DestroySession hapus sesi + cookie.
func DestroySession(db *sql.DB, w http.ResponseWriter, r *http.Request, cfg CookieConfig) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		db.Exec(`DELETE FROM sessions WHERE token=?`, hashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: cfg.Secure, SameSite: sameSite(cfg.SameSite),
	})
}

// CookieConfig atribut cookie sesi.
type CookieConfig struct {
	Secure   bool
	SameSite string // "lax" | "none"
}

func sameSite(s string) http.SameSite {
	if s == "none" {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

// User identitas login.
type User struct {
	ID    int64
	Email string
	Name  string
	Role  string
}

type ctxKey struct{}

// FromContext ambil user dari context (selalu ada di handler yang di-guard).
func FromContext(ctx context.Context) User {
	u, _ := ctx.Value(ctxKey{}).(User)
	return u
}

// SessionTokenRaw ambil token mentah sesi dari request (untuk CSRF check).
func sessionRawToken(r *http.Request) string {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

// Resolve cari user dari cookie sesi (tidak mengubah response).
func Resolve(db *sql.DB, r *http.Request) (User, string, bool) {
	raw := sessionRawToken(r)
	if raw == "" {
		return User{}, "", false
	}
	var u User
	var csrf string
	err := db.QueryRow(`SELECT u.id, u.email, u.name, u.role, s.csrf_token
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token=? AND s.expires_at > datetime('now')`, hashToken(raw)).
		Scan(&u.ID, &u.Email, &u.Name, &u.Role, &csrf)
	if err != nil {
		return User{}, "", false
	}
	return u, raw, true
}

// checkCSRF validasi header X-CSRF-Token terhadap sesi (stateless).
func checkCSRF(r *http.Request, csrf string) bool {
	got := r.Header.Get("X-CSRF-Token")
	if got == "" {
		got = r.PostFormValue("csrf")
	}
	return got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(csrf)) == 1
}

// Guard middleware: isi context user + cek CSRF untuk method tulis.
// onFail dipanggil (bukan redirect bawaan) supaya pemanggil menentukan bentuk
// respons (Inertia Redirect vs JSON 401).
func Guard(db *sql.DB, cfg CookieConfig, requireAdmin bool, next func(w http.ResponseWriter, r *http.Request, u User), onFail func(w http.ResponseWriter, r *http.Request, reason string)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _, ok := Resolve(db, r)
		if !ok {
			onFail(w, r, "unauthenticated")
			return
		}
		if requireAdmin && u.Role != "admin" {
			onFail(w, r, "forbidden")
			return
		}
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if !checkCSRF(r, CSRFToken(sessionRawToken(r))) {
				onFail(w, r, "csrf")
				return
			}
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)), u)
	})
}

// IsInertiaXHR true bila request datang dari Inertia (XHR) — dipakai onFail
// untuk memilih 401/403 JSON vs redirect.
func IsInertiaXHR(r *http.Request) bool {
	return r.Header.Get("X-Inertia") != "" || strings.HasPrefix(r.Header.Get("X-Requested-With"), "XMLHttpRequest")
}
