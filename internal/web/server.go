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
	s := &Server{DB: db, Cfg: cfg, Mail: ml, I: i, Version: version, Images: newR2Store(cfg.R2)}
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
	m.HandleFunc("GET /keranjang", func(w http.ResponseWriter, r *http.Request) {
		// isi keranjang hidup di localStorage klien — halaman cuma wadah
		s.Page(w, r, "Public/Cart", inertia.Props{})
	})
	m.HandleFunc("GET /privacy", func(w http.ResponseWriter, r *http.Request) {
		s.Page(w, r, "Public/Policy", inertia.Props{"kind": "privacy", "content": getSetting(s.DB, "content.privacy")})
	})
	m.HandleFunc("GET /ketentuan", func(w http.ResponseWriter, r *http.Request) {
		s.Page(w, r, "Public/Policy", inertia.Props{"kind": "terms", "content": getSetting(s.DB, "content.terms")})
	})

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

	// dev: masuk sebagai akun mana pun (hanya bila STORE_DEV_LOGIN=1)
	if s.Cfg.DevLogin {
		m.HandleFunc("GET /dev/login", s.devLoginPage)
		m.HandleFunc("POST /dev/login", s.devLoginAs)
	}

	// login tanpa password (tautan + kode email)
	m.Handle("GET /login/link", s.limited("magic", http.HandlerFunc(s.loginLinkPage)))
	m.Handle("POST /login/link", s.limited("magic", http.HandlerFunc(s.loginLinkSend)))
	m.HandleFunc("GET /login/link/check", s.loginLinkCheck)
	m.HandleFunc("GET /login/link/{token}", s.loginLinkConsume)
	m.Handle("POST /login/link/verify", s.limited("magic", http.HandlerFunc(s.loginLinkVerify)))

	// verifikasi email (login sudah, belum wajib verified)
	m.Handle("GET /verify-email", s.guardAuth(s.verifyEmailPage))
	m.Handle("POST /verify-email", s.guardAuth(s.verifyEmailVerify))
	m.Handle("POST /verify-email/resend", s.guardAuth(s.verifyEmailResend))

	// akun (Guard user + wajib email terverifikasi)
	m.Handle("GET /dashboard", s.guardVerified(s.dashboard))
	m.Handle("GET /profil", s.guardVerified(s.profilePage))
	m.Handle("POST /profil", s.guardVerified(s.updateProfile))
	m.Handle("POST /profil/password", s.guardVerified(s.changePassword))
	m.Handle("GET /pesanan", s.guardVerified(s.ordersPage))
	m.Handle("GET /pesanan/{order_no}", s.guardVerified(s.orderDetailPage))
	m.Handle("GET /tiket", s.guardVerified(s.ticketsPage))
	m.Handle("POST /tiket", s.guardVerified(s.createTicket))
	m.Handle("GET /tiket/{id}", s.guardVerified(s.ticketDetailPage))
	m.Handle("POST /tiket/{id}", s.guardVerified(s.replyTicket))

	// checkout & pembayaran (wajib login terverifikasi - produk dikirim ke
	// email yang terbukti dimiliki pembeli)
	m.Handle("POST /checkout", s.limited("checkout", s.guardVerified(func(w http.ResponseWriter, r *http.Request, _ auth.User) {
		s.checkout(w, r)
	})))
	m.Handle("GET /checkout", s.limited("checkout", s.guardVerified(func(w http.ResponseWriter, r *http.Request, _ auth.User) {
		s.Page(w, r, "Public/Checkout", inertia.Props{})
	})))
	m.HandleFunc("GET /pay/{order_no}", s.payPage)
	m.Handle("POST /pay/{order_no}", s.limited("checkout", http.HandlerFunc(s.payNow)))
	m.Handle("POST /callback/payment", s.limited("callback", http.HandlerFunc(s.paymentCallback)))
	m.Handle("GET /dl/{order_no}/{item}", s.limited("download", http.HandlerFunc(s.download)))

	// admin (role admin)
	m.Handle("GET /admin", s.guardAdmin(s.adminDashboard))
	m.Handle("GET /admin/users", s.guardAdmin(s.adminUsers))
	m.Handle("GET /admin/orders", s.guardAdmin(s.adminOrders))
	m.Handle("POST /admin/orders/{order_no}/paid", s.guardAdmin(s.adminOrderPaid))
	m.Handle("POST /admin/orders/{order_no}/resend", s.guardAdmin(s.adminOrderResend))
	m.Handle("GET /admin/produk", s.guardAdmin(s.adminProducts))
	m.Handle("POST /admin/produk", s.guardAdmin(s.adminProductSave))
	m.Handle("POST /admin/produk/{id}", s.guardAdmin(s.adminProductSave))
	m.Handle("POST /admin/produk/{id}/keys", s.guardAdmin(s.adminProductKeys))
	m.Handle("POST /admin/produk/{id}/images", s.guardAdmin(s.adminProductImages))
	m.Handle("POST /admin/produk/{id}/images/{image_id}/delete", s.guardAdmin(s.adminProductImageDelete))
	m.Handle("GET /admin/orders/{order_no}", s.guardAdmin(s.adminOrderDetail))

	// setting admin
	m.Handle("GET /admin/setting", s.guardAdmin(s.adminSettingPage))
	m.Handle("POST /admin/setting/smtp", s.guardAdmin(s.adminSettingSMTP))
	m.Handle("POST /admin/setting/smtp/test", s.guardAdmin(s.adminSettingSMTPTest))
	m.Handle("POST /admin/setting/brand", s.guardAdmin(s.adminSettingBrand))
	m.Handle("POST /admin/setting/theme", s.guardAdmin(s.adminSettingTheme))
	m.Handle("POST /admin/setting/content", s.guardAdmin(s.adminSettingContent))
	m.Handle("POST /admin/setting/seo", s.guardAdmin(s.adminSettingSEO))
	m.Handle("POST /admin/produk/{id}/toggle", s.guardAdmin(s.adminProductToggle))
	m.Handle("GET /admin/maillog", s.guardAdmin(s.adminMailLog))
	m.Handle("GET /admin/tiket", s.guardAdmin(s.adminTickets))
	m.Handle("GET /admin/tiket/{id}", s.guardAdmin(s.adminTicketDetailPage))
	m.Handle("POST /admin/tiket/{id}", s.guardAdmin(s.adminTicketReply))
	m.Handle("POST /admin/tiket/{id}/close", s.guardAdmin(s.adminTicketClose))
	m.Handle("GET /admin/gateway", s.guardAdmin(s.adminGateways))
	m.Handle("POST /admin/gateway/config", s.guardAdmin(s.adminGatewayConfig))
	m.Handle("POST /admin/gateway/activate", s.guardAdmin(s.adminGatewayActivate))

	// aset Vite (mode shell-Go); produksi utama aset disajikan CF Pages.
	m.HandleFunc("GET /assets/{path...}", s.serveAssets)

	// gambar produk dari bucket R2 (proxy bila tanpa domain publik)
	m.HandleFunc("GET /images/{key...}", s.productImageProxy)

	s.Mux = m
}

// serveAssets sajikan aset build Vite dari web/dist (immutable).
func (s *Server) serveAssets(w http.ResponseWriter, r *http.Request) {
	rel := filepath.Clean(r.PathValue("path"))
	if strings.Contains(rel, "..") {
		http.NotFound(w, r)
		return
	}
	full := filepath.Join("web", "dist", "assets", rel)
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
