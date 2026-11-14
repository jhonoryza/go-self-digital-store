package web

// Server: route registration (method-pattern mux), rantai middleware, dan
// penyajian aset Vite untuk mode shell-Go.

import (
	"database/sql"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	"github.com/jhonoryza/go-self-digital-store/internal/config"
	"github.com/jhonoryza/go-self-digital-store/internal/mail"
)

// New bangun Server lengkap beserta rantai middleware:
// recover -> secHeaders -> cors -> originCheck -> inertia -> mux.
func New(db *sql.DB, cfg config.Config, ml *mail.Mailer, i *inertia.Inertia, version string) *Server { //nolint:revive
	s := &Server{DB: db, Cfg: cfg, Mail: ml, I: i, Version: version}
	s.routes()
	s.Mux = recover500(s.secHeaders(s.cors(s.originCheck(s.inertiaMiddleware(s.Mux)))))
	return s
}

// originCheck pertahanan CSRF tingkat pertama: method tulis dari browser harus
// ber-Origin sama dengan API atau termasuk allowlist CORS. Non-browser
// (webhook gateway, curl) tanpa Origin lolos — mereka tidak membawa cookie.
func (s *Server) originCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			origin := r.Header.Get("Origin")
			if origin != "" && !s.originAllowed(r, origin) {
				jsonError(w, http.StatusForbidden, "origin tidak diizinkan")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(r *http.Request, origin string) bool {
	for _, o := range s.Cfg.CorsOrigins {
		if origin == o {
			return true
		}
	}
	// dev tanpa CORS: izinkan origin localhost (vite proxy meneruskan Origin asli)
	if len(s.Cfg.CorsOrigins) == 0 {
		if u, err := neturl.Parse(origin); err == nil &&
			(u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1") {
			return true
		}
	}
	host := r.Host
	if s.Cfg.BaseURL != "" {
		host = strings.TrimPrefix(strings.TrimPrefix(s.Cfg.BaseURL, "https://"), "http://")
	}
	return strings.TrimPrefix(origin, "https://") == host || strings.TrimPrefix(origin, "http://") == host
}

// routes daftar seluruh route. Per-milestone: M2 auth, M3 checkout, M4 delivery,
// M5 akun, M6 admin.
func (s *Server) routes() {
	m := http.NewServeMux()

	// publik
	m.HandleFunc("GET /{$}", s.landing)
	m.HandleFunc("GET /produk", s.productsList)
	m.HandleFunc("GET /produk/{slug}", s.productDetail)
	m.HandleFunc("GET /privacy", func(w http.ResponseWriter, r *http.Request) { s.Page(w, r, "Public/Policy", inertia.Props{"kind": "privacy"}) })
	m.HandleFunc("GET /ketentuan", func(w http.ResponseWriter, r *http.Request) { s.Page(w, r, "Public/Policy", inertia.Props{"kind": "terms"}) })

	// health
	m.HandleFunc("GET /healthz", s.healthz)

	// auth
	m.Handle("GET /login", s.limited("login", http.HandlerFunc(s.loginPage)))
	m.Handle("POST /login", s.limited("login", http.HandlerFunc(s.login)))
	m.Handle("GET /register", s.limited("register", http.HandlerFunc(s.registerPage)))
	m.Handle("POST /register", s.limited("register", http.HandlerFunc(s.register)))
	m.Handle("POST /logout", s.guardAuth(s.logout))
	m.Handle("GET /lupa-password", s.limited("reset", http.HandlerFunc(s.forgotPage)))
	m.Handle("POST /lupa-password", s.limited("reset", http.HandlerFunc(s.forgotPassword)))
	m.HandleFunc("GET /reset-password", s.resetPage)
	m.Handle("POST /reset-password", s.limited("reset", http.HandlerFunc(s.resetPassword)))

	// akun (Guard user)
	m.Handle("GET /dashboard", s.guardAuth(s.dashboard))
	m.Handle("GET /profil", s.guardAuth(s.profilePage))
	m.Handle("POST /profil", s.guardAuth(s.updateProfile))
	m.Handle("POST /profil/password", s.guardAuth(s.changePassword))

	// checkout & pembayaran
	m.Handle("POST /checkout", s.limited("checkout", http.HandlerFunc(s.checkout)))
	m.HandleFunc("GET /pay/{order_no}", s.payPage)
	m.Handle("POST /pay/{order_no}", s.limited("checkout", http.HandlerFunc(s.payNow)))
	m.Handle("POST /callback/payment", s.limited("callback", http.HandlerFunc(s.paymentCallback)))

	// aset Vite (mode shell-Go); produksi utama aset disajikan CF Pages.
	m.HandleFunc("GET /assets/{path...}", s.serveAssets)

	s.Mux = m
}

// serveAssets sajikan aset build Vite dari web/dist (immutable).
func (s *Server) serveAssets(w http.ResponseWriter, r *http.Request) {
	rel := filepath.Clean(r.PathValue("path"))
	if strings.Contains(rel, "..") {
		http.NotFound(w, r)
		return
	}
	full := filepath.Join("web", "dist", rel)
	f, err := os.Open(full)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31556952, immutable")
	http.ServeContent(w, r, st.Name(), st.ModTime(), f)
}

// authOnFail respons gagal Guard: Inertia XHR -> X-Inertia-Location (redirect
// penuh via klien), biasa -> 302.
func (s *Server) authOnFail(target string, admin bool) func(http.ResponseWriter, *http.Request, string) {
	return func(w http.ResponseWriter, r *http.Request, reason string) {
		switch reason {
		case "csrf":
			if auth.IsInertiaXHR(r) {
				jsonError(w, http.StatusForbidden, `{"error":"sesi kedaluwarsa, muat ulang halaman"}`)
				return
			}
			http.Error(w, "sesi kedaluwarsa, muat ulang halaman", http.StatusForbidden)
			return
		case "forbidden":
			if auth.IsInertiaXHR(r) {
				jsonError(w, http.StatusForbidden, `{"error":"akses ditolak"}`)
				return
			}
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		// unauthenticated
		if auth.IsInertiaXHR(r) {
			w.Header().Set("X-Inertia-Location", target)
			w.WriteHeader(http.StatusConflict)
			return
		}
		http.Redirect(w, r, target, http.StatusFound)
	}
}
