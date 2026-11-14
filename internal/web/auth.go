package web

// Auth: register, login, logout, lupa/reset password.
// Validasi gagal → SetValidationErrors + render ulang komponen (Inertia memperbarui
// halaman dengan props.errors tanpa pindah halaman).

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	"github.com/jhonoryza/go-self-digital-store/internal/mail"
)

// ---- login lockout in-memory (5 gagal → kunci 10 menit, kunci email|IP) ----

type failEntry struct {
	count int
	until time.Time
}

var loginFails = map[string]failEntry{}

func loginBlocked(email, ip string) bool {
	f, ok := loginFails[email+"|"+ip]
	return ok && time.Now().Before(f.until)
}

func loginFail(email, ip string) {
	now := time.Now()
	key := email + "|" + ip
	f := loginFails[key]
	f.count++
	if f.count >= 5 {
		f.until = now.Add(10 * time.Minute)
	}
	loginFails[key] = f
}

func loginOK(email, ip string) {
	delete(loginFails, email+"|"+ip)
}

// ---- GET halaman ----

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := auth.Resolve(s.DB, r); ok {
		s.Redirect(w, r, "/dashboard")
		return
	}
	s.Page(w, r, "Auth/Login", inertia.Props{"flash": flashFromQuery(r)})
}

func (s *Server) registerPage(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := auth.Resolve(s.DB, r); ok {
		s.Redirect(w, r, "/dashboard")
		return
	}
	s.Page(w, r, "Auth/Register", inertia.Props{"flash": flashFromQuery(r)})
}

func (s *Server) forgotPage(w http.ResponseWriter, r *http.Request) {
	s.Page(w, r, "Auth/ForgotPassword", inertia.Props{"flash": flashFromQuery(r)})
}

func (s *Server) resetPage(w http.ResponseWriter, r *http.Request) {
	s.Page(w, r, "Auth/ResetPassword", inertia.Props{
		"token": r.URL.Query().Get("token"),
		"flash": flashFromQuery(r),
	})
}

// ---- POST aksi ----

// validEmail cek format longgar (ada @ dan domain).
func validEmail(e string) bool {
	at := strings.Index(e, "@")
	return at > 0 && at < len(e)-1 && !strings.ContainsAny(e, " \t\r\n")
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if ipLimited("register", s.clientIP(r)) {
		jsonError(w, http.StatusTooManyRequests, "terlalu banyak percobaan")
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.PostFormValue("email")))
	name := strings.TrimSpace(r.PostFormValue("name"))
	password := r.PostFormValue("password")

	errs := inertia.ValidationErrors{}
	if !validEmail(email) {
		errs["email"] = "email tidak valid"
	}
	if name == "" {
		errs["name"] = "nama wajib diisi"
	}
	if len(password) < 8 {
		errs["password"] = "password minimal 8 karakter"
	}
	var exists int
	s.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE email=?`, email).Scan(&exists)
	if exists > 0 {
		errs["email"] = "email sudah terdaftar"
	}
	if len(errs) > 0 {
		s.renderWithErrors(w, r, "Auth/Register", nil, errs)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	res, err := s.DB.Exec(`INSERT INTO users (email, name, password) VALUES (?,?,?)`, email, name, hash)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	uid, _ := res.LastInsertId()
	if _, err := auth.StartSession(s.DB, w, uid, s.cookieCfg()); err != nil {
		jsonError(w, http.StatusInternalServerError, "gagal membuat sesi")
		return
	}
	s.Redirect(w, r, "/dashboard")
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.PostFormValue("email")))
	password := r.PostFormValue("password")
	ip := s.clientIP(r)
	if ipLimited("login", ip) {
		jsonError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, tunggu sebentar")
		return
	}
	if loginBlocked(email, ip) {
		jsonError(w, http.StatusTooManyRequests, "terlalu banyak kegagalan — coba lagi dalam 10 menit")
		return
	}

	var u struct {
		ID    int64
		Pass  string
	}
	err := s.DB.QueryRow(`SELECT id, password FROM users WHERE email=?`, email).Scan(&u.ID, &u.Pass)
	if err == sql.ErrNoRows {
		// timing equalizer: hash-verify dummy agar unknown-email tak terbedakan
		dummy, _ := auth.HashPassword("timing-equalizer-dummy")
		auth.VerifyPassword(dummy, password)
		loginFail(email, ip)
		s.renderWithErrors(w, r, "Auth/Login", nil, inertia.ValidationErrors{"email": "email atau password salah"})
		return
	} else if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	if !auth.VerifyPassword(u.Pass, password) {
		loginFail(email, ip)
		s.renderWithErrors(w, r, "Auth/Login", nil, inertia.ValidationErrors{"email": "email atau password salah"})
		return
	}
	loginOK(email, ip)
	if _, err := auth.StartSession(s.DB, w, u.ID, s.cookieCfg()); err != nil {
		jsonError(w, http.StatusInternalServerError, "gagal membuat sesi")
		return
	}

	// admin → konsol admin; user → dashboard
	var role string
	s.DB.QueryRow(`SELECT role FROM users WHERE id=?`, u.ID).Scan(&role)
	if role == "admin" {
		s.Redirect(w, r, "/admin")
		return
	}
	s.Redirect(w, r, "/dashboard")
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request, _ auth.User) {
	auth.DestroySession(s.DB, w, r, s.cookieCfg())
	s.Redirect(w, r, "/")
}

func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	if ipLimited("reset", s.clientIP(r)) {
		jsonError(w, http.StatusTooManyRequests, "terlalu banyak percobaan")
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.PostFormValue("email")))

	// respons selalu sukses (anti enumerasi email)
	var uid int64
	err := s.DB.QueryRow(`SELECT id FROM users WHERE email=?`, email).Scan(&uid)
	if err == nil {
		token := auth.Token()
		s.DB.Exec(`INSERT INTO password_resets (email, token, expires_at) VALUES (?,?, datetime('now', '+1 hour'))`, email, token)
		link := s.baseURL(r) + "/reset-password?token=" + token
		if s.Mail.Enabled() {
			go func() {
				body := mail.Style(s.Cfg.AppName,
					`<p>Halo,</p><p>Ada permintaan reset password untuk akun Anda. Tautan berlaku 1 jam:</p>
					 <p>`+mail.Button(link, "Reset Password")+`</p>
					 <p style="color:#71717a;">Bukan Anda? Abaikan email ini — password tidak berubah.</p>`)
				s.Mail.Send(email, "Reset password — "+s.Cfg.AppName, body)
			}()
		}
	}
	s.flashOK(w, r, "/login", "Bila email terdaftar, tautan reset sudah dikirim.")
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PostFormValue("token"))
	password := r.PostFormValue("password")
	confirm := r.PostFormValue("password_confirmation")

	errs := inertia.ValidationErrors{}
	if len(password) < 8 {
		errs["password"] = "password minimal 8 karakter"
	}
	if password != confirm {
		errs["password_confirmation"] = "konfirmasi password tidak cocok"
	}
	var email string
	s.DB.QueryRow(`SELECT email FROM password_resets WHERE token=? AND expires_at > datetime('now')`, token).Scan(&email)
	if email == "" {
		errs["token"] = "tautan reset tidak berlaku atau kedaluwarsa"
	}
	if len(errs) > 0 {
		s.renderWithErrors(w, r, "Auth/ResetPassword", inertia.Props{"token": token}, errs)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	if _, err := s.DB.Exec(`UPDATE users SET password=? WHERE email=?`, hash, email); err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	// token sekali pakai + putus semua sesi lama akun itu
	s.DB.Exec(`DELETE FROM password_resets WHERE email=?`, email)
	s.DB.Exec(`DELETE FROM sessions WHERE user_id=(SELECT id FROM users WHERE email=?)`, email)
	s.flashOK(w, r, "/login", "Password diganti — silakan masuk.")
}

// ---- util validasi ----

// renderWithErrors render ulang komponen dengan props.errors terisi (alur
// validasi tanpa redirect).
func (s *Server) renderWithErrors(w http.ResponseWriter, r *http.Request, component string, props inertia.Props, errs inertia.ValidationErrors) {
	ctx := inertia.SetValidationErrors(r.Context(), errs)
	s.I.Render(w, r.WithContext(ctx), component, props)
}

// cookieCfg atribut cookie sesi dari konfigurasi.
func (s *Server) cookieCfg() auth.CookieConfig {
	return auth.CookieConfig{Secure: s.Cfg.CookieSecure, SameSite: s.Cfg.CookieSameSite}
}
