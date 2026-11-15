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

// devLoginPage GET /dev/login — form user_id polos. Sengaja TIDAK
// mengenumerasi akun: halaman dev tetap tidak bocor daftar user bila flag
// tersalahkan aktif di host yang terlihat orang lain.
func (s *Server) devLoginPage(w http.ResponseWriter, r *http.Request) {
	if !s.Cfg.DevLogin {
		http.NotFound(w, r)
		return
	}
	s.Page(w, r, "Dev/LoginAs", inertia.Props{"flash": flashFromQuery(r)})
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
