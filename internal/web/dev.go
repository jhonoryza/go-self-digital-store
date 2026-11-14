package web

// Dev login-as: masuk sebagai user/admin mana pun tanpa password — khusus
// dev environment (STORE_DEV_LOGIN=1). Route tidak terdaftar bila flag mati,
// jadi produksi tidak pernah punya jalur ini.

import (
	"database/sql"
	"net/http"
	"strconv"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
)

// devAccount baris akun untuk halaman dev login.
type devAccount struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// devAccounts seluruh akun, admin dulu lalu user (nama naik).
func (s *Server) devAccounts() []devAccount {
	rows, err := s.DB.Query(`SELECT id, email, name, role FROM users ORDER BY role='user', name, id`)
	if err != nil {
		logErr("dev accounts: %v", err)
		return nil
	}
	defer rows.Close()
	var out []devAccount
	for rows.Next() {
		var a devAccount
		if rows.Scan(&a.ID, &a.Email, &a.Name, &a.Role) == nil {
			out = append(out, a)
		}
	}
	return out
}

// devLoginPage GET /dev/login — daftar akun + tombol masuk sebagai.
func (s *Server) devLoginPage(w http.ResponseWriter, r *http.Request) {
	if !s.Cfg.DevLogin {
		http.NotFound(w, r)
		return
	}
	s.Page(w, r, "Dev/LoginAs", inertia.Props{"accounts": s.devAccounts()})
}

// devLoginAs POST /dev/login — buat sesi untuk user_id yang dipilih.
// Tanpa cek CSRF: dev-only, tidak membawa aksi berbahaya, dan halamannya
// dilihat guest (csrf_token props hanya ada untuk sesi yang sudah login).
func (s *Server) devLoginAs(w http.ResponseWriter, r *http.Request) {
	if !s.Cfg.DevLogin {
		http.NotFound(w, r)
		return
	}
	uid, err := strconv.ParseInt(r.PostFormValue("user_id"), 10, 64)
	if err != nil {
		s.flashErr(w, r, "/dev/login", "user_id tidak valid.")
		return
	}
	var role string
	err = s.DB.QueryRow(`SELECT role FROM users WHERE id=?`, uid).Scan(&role)
	if err == sql.ErrNoRows {
		s.flashErr(w, r, "/dev/login", "Akun tidak ditemukan.")
		return
	} else if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	if _, err := auth.StartSession(s.DB, w, uid, s.cookieCfg()); err != nil {
		jsonError(w, http.StatusInternalServerError, "gagal membuat sesi")
		return
	}
	audit(s.DB, "dev.login_as", strconv.FormatInt(uid, 10), "role "+role)
	target := "/dashboard"
	if role == "admin" {
		target = "/admin"
	}
	s.flashOK(w, r, target, "Masuk sebagai "+strconv.FormatInt(uid, 10)+" (dev).")
}
