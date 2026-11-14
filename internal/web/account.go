package web

// Area akun (Guard): profil, ganti password, dashboard ringkas (M5 melengkapi).

import (
	"net/http"
	"strings"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
)

// guardAuth helper: bungkus handler dengan Guard(user).
func (s *Server) guardAuth(next func(http.ResponseWriter, *http.Request, auth.User)) http.Handler {
	return auth.Guard(s.DB, s.cookieCfg(), false, next, s.authOnFail("/login", false))
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request, u auth.User) {
	s.Page(w, r, "Account/Dashboard", inertia.Props{"flash": flashFromQuery(r)})
}

func (s *Server) profilePage(w http.ResponseWriter, r *http.Request, u auth.User) {
	s.Page(w, r, "Account/Profile", inertia.Props{"flash": flashFromQuery(r)})
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request, u auth.User) {
	name := strings.TrimSpace(r.PostFormValue("name"))
	email := strings.ToLower(strings.TrimSpace(r.PostFormValue("email")))

	errs := inertia.ValidationErrors{}
	if name == "" {
		errs["name"] = "nama wajib diisi"
	}
	if !validEmail(email) {
		errs["email"] = "email tidak valid"
	}
	var exists int
	s.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE email=? AND id<>?`, email, u.ID).Scan(&exists)
	if exists > 0 {
		errs["email"] = "email sudah dipakai akun lain"
	}
	if len(errs) > 0 {
		s.renderWithErrors(w, r, "Account/Profile", nil, errs)
		return
	}
	if _, err := s.DB.Exec(`UPDATE users SET name=?, email=? WHERE id=?`, name, email, u.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	audit(s.DB, "profil.update", u.Email, email)
	s.flashOK(w, r, "/profil", "Profil diperbarui.")
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request, u auth.User) {
	current := r.PostFormValue("current_password")
	password := r.PostFormValue("password")
	confirm := r.PostFormValue("password_confirmation")

	errs := inertia.ValidationErrors{}
	var hash string
	err := s.DB.QueryRow(`SELECT password FROM users WHERE id=?`, u.ID).Scan(&hash)
	if err != nil || !auth.VerifyPassword(hash, current) {
		errs["current_password"] = "password saat ini salah"
	}
	if len(password) < 8 {
		errs["password"] = "password baru minimal 8 karakter"
	}
	if password != confirm {
		errs["password_confirmation"] = "konfirmasi password tidak cocok"
	}
	if len(errs) > 0 {
		s.renderWithErrors(w, r, "Account/Profile", nil, errs)
		return
	}
	newHash, err := auth.HashPassword(password)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	if _, err := s.DB.Exec(`UPDATE users SET password=? WHERE id=?`, newHash, u.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	// putus sesi lain (sesi saat ini dipertahankan)
	auth.DeleteOtherSessions(s.DB, r, u.ID)
	s.flashOK(w, r, "/profil", "Password diganti.")
}
