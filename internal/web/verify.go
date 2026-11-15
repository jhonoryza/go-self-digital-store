package web

// Verifikasi email: saat daftar, OTP 6 digit dikirim ke email; akun belum
// bisa masuk area dashboard sebelum memasukkan OTP di halaman verifikasi.
// Checkout juga wajib login-terverifikasi agar produk selalu terkirim ke
// email yang terbukti dimiliki pembeli.

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	"github.com/jhonoryza/go-self-digital-store/internal/mail"
)

const otpTTLMinutes = 15
const otpResendSeconds = 60

// hashOTP OTP disimpan sebagai sha256 — kebocoran DB tidak langsung memberi kode.
func hashOTP(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// generateOTP 6 digit acak (leading zero dipertahankan).
func generateOTP() string {
	b := make([]byte, 4)
	rand.Read(b)
	v := (int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])) % 1000000
	return fmt.Sprintf("%06d", v)
}

// sendEmailOTP buat + kirim OTP baru ke email user; kembalikan kode plaintext
// (handler mengabaikannya — test e2e memakai untuk memverifikasi alur).
func (s *Server) sendEmailOTP(u auth.User) (string, error) {
	code := generateOTP()
	_, err := s.DB.Exec(`INSERT INTO email_otps (user_id, code, expires_at, last_sent_at) VALUES (?,?,datetime('now', ?),datetime('now'))
		ON CONFLICT(user_id) DO UPDATE SET code=excluded.code, expires_at=excluded.expires_at, last_sent_at=excluded.last_sent_at`,
		u.ID, hashOTP(code), fmt.Sprintf("+%d minutes", otpTTLMinutes))
	if err != nil {
		return "", err
	}
	body := mail.Style(s.appName(), fmt.Sprintf(
		`<p>Halo,</p><p>Kode verifikasi email Anda:</p>`+
			`<p style="margin:16px 0;"><span style="font-family:Arial,Helvetica,sans-serif;font-size:28px;font-weight:bold;letter-spacing:8px;">%s</span></p>`+
			`<p>Kode berlaku %d menit. Bila Anda tidak merasa mendaftar, abaikan email ini.</p>`,
		code, otpTTLMinutes))
	go s.Mail.Send(u.Email, "Kode verifikasi - "+s.appName(), body)
	return code, nil
}

// verifiedRedirect target setelah verifikasi sesuai role.
func verifiedRedirect(role string) string {
	if role == "admin" {
		return "/admin"
	}
	return "/dashboard"
}

// verifyEmailPage GET /verify-email — halaman input OTP.
func (s *Server) verifyEmailPage(w http.ResponseWriter, r *http.Request, u auth.User) {
	if u.EmailVerifiedAt != "" {
		s.Redirect(w, r, verifiedRedirect(u.Role))
		return
	}
	s.Page(w, r, "Auth/VerifyEmail", inertia.Props{
		"email":     u.Email,
		"sent_otp":  getSetting(s.DB, "smtp.host") != "" || s.Cfg.SMTP.Host != "",
		"resend_in": s.otpResendIn(u.ID),
		"flash":     flashFromQuery(r),
	})
}

// otpResendIn detik tersisa sebelum OTP boleh dikirim ulang (0 = boleh).
func (s *Server) otpResendIn(userID int64) int {
	var wait int
	s.DB.QueryRow(`SELECT CAST(MAX(0, strftime('%s', last_sent_at) + ?1 - strftime('%s','now')) AS INTEGER)
		FROM email_otps WHERE user_id=?`, otpResendSeconds, userID).Scan(&wait)
	return wait
}

// verifyEmailVerify POST /verify-email — cocokkan OTP lalu tandai verified.
func (s *Server) verifyEmailVerify(w http.ResponseWriter, r *http.Request, u auth.User) {
	if u.EmailVerifiedAt != "" {
		s.Redirect(w, r, verifiedRedirect(u.Role))
		return
	}
	code := strings.TrimSpace(r.PostFormValue("code"))
	if len(code) != 6 {
		s.renderWithErrors(w, r, "Auth/VerifyEmail", inertia.Props{
			"email": u.Email, "resend_in": s.otpResendIn(u.ID),
		}, inertia.ValidationErrors{"code": "Kode harus 6 digit."})
		return
	}
	var stored string
	var expired int
	err := s.DB.QueryRow(`SELECT code, expires_at <= datetime('now') FROM email_otps WHERE user_id=?`, u.ID).
		Scan(&stored, &expired)
	if err == sql.ErrNoRows {
		s.renderWithErrors(w, r, "Auth/VerifyEmail", inertia.Props{
			"email": u.Email, "resend_in": s.otpResendIn(u.ID),
		}, inertia.ValidationErrors{"code": "Kode belum dibuat - kirim ulang OTP."})
		return
	} else if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	if expired == 1 {
		s.renderWithErrors(w, r, "Auth/VerifyEmail", inertia.Props{
			"email": u.Email, "resend_in": s.otpResendIn(u.ID),
		}, inertia.ValidationErrors{"code": "Kode kedaluwarsa - kirim ulang OTP."})
		return
	}
	if subtle.ConstantTimeCompare([]byte(hashOTP(code)), []byte(stored)) != 1 {
		s.renderWithErrors(w, r, "Auth/VerifyEmail", inertia.Props{
			"email": u.Email, "resend_in": s.otpResendIn(u.ID),
		}, inertia.ValidationErrors{"code": "Kode salah."})
		return
	}
	if _, err := s.DB.Exec(`UPDATE users SET email_verified_at=datetime('now') WHERE id=?`, u.ID); err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	s.DB.Exec(`DELETE FROM email_otps WHERE user_id=?`, u.ID)
	audit(s.DB, "user.verify_email", u.Email, "")
	s.flashOK(w, r, verifiedRedirect(u.Role), "Email terverifikasi - selamat datang!")
}

// verifyEmailResend POST /verify-email/resend — kirim OTP baru (cooldown 60s).
func (s *Server) verifyEmailResend(w http.ResponseWriter, r *http.Request, u auth.User) {
	if u.EmailVerifiedAt != "" {
		s.Redirect(w, r, verifiedRedirect(u.Role))
		return
	}
	if wait := s.otpResendIn(u.ID); wait > 0 {
		s.flashErr(w, r, "/verify-email", fmt.Sprintf("Tunggu %d detik sebelum kirim ulang.", wait))
		return
	}
	if _, err := s.sendEmailOTP(u); err != nil {
		logErr("kirim OTP %s: %v", u.Email, err)
		s.flashErr(w, r, "/verify-email", "Gagal mengirim OTP - coba lagi.")
		return
	}
	s.flashOK(w, r, "/verify-email", "OTP baru dikirim ke "+u.Email+".")
}

// guardVerified auth + wajib email terverifikasi; gagal diarahkan ke
// /verify-email (409 + X-Inertia-Location untuk XHR).
func (s *Server) guardVerified(next func(http.ResponseWriter, *http.Request, auth.User)) http.Handler {
	return auth.Guard(s.DB, s.cookieCfg(), false, func(w http.ResponseWriter, r *http.Request, u auth.User) {
		if u.EmailVerifiedAt == "" {
			s.authOnFail("/verify-email", false)(w, r, "unauthenticated")
			return
		}
		next(w, r, u)
	}, s.authOnFail("/login", false))
}
