package web

// Helper lintas handler: log, settings, audit, rupiah, token, render + redirect.

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	"github.com/jhonoryza/go-self-digital-store/internal/config"
	"github.com/jhonoryza/go-self-digital-store/internal/mail"
)

func logErr(f string, v ...any) { log.Printf("[web] "+f, v...) }

// getSetting / setSetting — key-value runtime (konfigurasi gateway dll).
func getSetting(db *sql.DB, key string) string {
	var v string
	db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	return v
}

func setSetting(db *sql.DB, key, value string) {
	db.Exec(`INSERT INTO settings (key, value) VALUES (?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
}

// audit catat aksi konsol admin.
func audit(db *sql.DB, action, target, detail string) {
	db.Exec(`INSERT INTO audit_log (action, target, detail) VALUES (?,?,?)`, action, target, detail)
}

// rupiah format IDR gaya Indonesia (1.250.000) — kelompok 3 digit dari kanan.
func rupiah(n int) string {
	neg := n < 0
	if neg {
		n = -n
	}
	digits := strconv.Itoa(n)
	var b strings.Builder
	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-Rp" + b.String()
	}
	return "Rp" + b.String()
}

// truncate potong string untuk pesan error/log.
func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// tok token acak n-byte hex (crypto/rand).
func tok(n int) string { return auth.Token()[:n*2] }

// baseURL absolute base untuk tautan email/unduhan; fallback request host.
func (s *Server) baseURL(r *http.Request) string {
	if s.Cfg.BaseURL != "" {
		return s.Cfg.BaseURL
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// Page render komponen Inertia dengan props bersama per-request:
// user (null bila guest) + csrf_token (untuk http adapter frontend).
func (s *Server) Page(w http.ResponseWriter, r *http.Request, component string, props inertia.Props) error {
	if props == nil {
		props = inertia.Props{}
	}
	if u, raw, ok := auth.Resolve(s.DB, r); ok {
		props["auth_user"] = map[string]any{"id": u.ID, "email": u.Email, "name": u.Name, "role": u.Role}
		props["csrf_token"] = auth.CSRFToken(raw)
	}
	return s.I.Render(w, r, component, props)
}

// Redirect Inertia-aware (303 pada XHR PUT/PATCH/DELETE ditangani gonertia).
func (s *Server) Redirect(w http.ResponseWriter, r *http.Request, url string) {
	s.I.Redirect(w, r, url)
}

// jsonError jawab error JSON untuk XHR.
func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}

// ---- flash sederhana via query param (?ok= / ?err=) — konsisten Inertia ----

func (s *Server) flashOK(w http.ResponseWriter, r *http.Request, url, msg string) {
	s.Redirect(w, r, withParam(url, "ok", msg))
}

func (s *Server) flashErr(w http.ResponseWriter, r *http.Request, url, msg string) {
	s.Redirect(w, r, withParam(url, "err", msg))
}

// withParam tambah satu query param pada URL.
func withParam(rawURL, key, val string) string {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	q.Set(key, val)
	u.RawQuery = q.Encode()
	return u.String()
}

// Server kumpulan dependensi aplikasi web.
type Server struct {
	DB      *sql.DB
	Cfg     config.Config
	Mail    *mail.Mailer
	I       *inertia.Inertia
	Version string // versi aset untuk X-Inertia-Version
	Mux     http.Handler
}
