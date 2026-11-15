package web

// Login tanpa password: pengguna memasukkan email, sistem mengirim SATU
// email berisi tautan ajaib (klik = langsung masuk) dan kode OTP 6 digit
// (diketik di halaman). Keduanya sekali pakai, TTL 15 menit, maks 5
// percobaan kode, dan otomatis menandai email terverifikasi karena
// kepemilikan email terbukti.

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	"github.com/jhonoryza/go-self-digital-store/internal/mail"
)

const loginTokenTTLMinutes = 15
const loginTokenMaxAttempts = 5

// loginLinkPage GET /login/link — form email (+ form kode setelah dikirim).
func (s *Server) loginLinkPage(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := auth.Resolve(s.DB, r); ok {
		s.Redirect(w, r, "/dashboard")
		return
	}
	s.Page(w, r, "Auth/MagicLink", inertia.Props{
		"email":   strings.TrimSpace(r.URL.Query().Get("email")),
		"sent_to": r.URL.Query().Get("sent"),
		"flash":   flashFromQuery(r),
	})
}

// createLoginTokens buat pasangan tautan + kode untuk satu email; kembalikan
// plaintext keduanya (handler mengabaikan — test e2e memakai).
func (s *Server) createLoginTokens(email string) (linkToken, code string, err error) {
	email = strings.ToLower(email)
	var userID int64
	if err = s.DB.QueryRow(`SELECT id FROM users WHERE email=?`, email).Scan(&userID); err != nil {
		return "", "", err
	}
	// rapikan token kedaluwarsa/terpakai milik email ini
	s.DB.Exec(`DELETE FROM login_tokens WHERE email=? AND (expires_at <= datetime('now') OR used_at IS NOT NULL)`, email)

	linkB := make([]byte, 32)
	rand.Read(linkB)
	linkToken = hex.EncodeToString(linkB)
	code = generateOTP()

	_, err = s.DB.Exec(`INSERT INTO login_tokens (user_id, email, token_hash, code_hash, expires_at)
		VALUES (?,?,?,?,datetime('now', ?))`,
		userID, email, hashOTP(linkToken), hashOTP(code),
		fmt.Sprintf("+%d minutes", loginTokenTTLMinutes))
	return linkToken, code, err
}

// loginLinkSend POST /login/link — kirim email; respons selalu generik agar
// keberadaan akun tidak bisa dipeleset.
func (s *Server) loginLinkSend(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.PostFormValue("email")))
	if !validEmail(email) {
		s.flashErr(w, r, "/login/link", "Email tidak valid.")
		return
	}

	var userID int64
	err := s.DB.QueryRow(`SELECT id FROM users WHERE email=?`, email).Scan(&userID)
	if err == nil {
		linkToken, code, err := s.createLoginTokens(email)
		if err == nil {
			link := s.baseURL(r) + "/login/link/" + linkToken
			body := mail.Style(s.appName(), fmt.Sprintf(
				`<p>Halo,</p><p>Klik tombol berikut untuk masuk tanpa password:</p>%s`+
					`<p style="margin:16px 0 0;">Atau masukkan kode ini di halaman masuk:</p>`+
					`<p style="margin:8px 0;"><span style="font-family:Arial,Helvetica,sans-serif;font-size:24px;font-weight:bold;letter-spacing:6px;">%s</span></p>`+
					`<p style="color:#71717a;">Tautan dan kode berlaku %d menit dan sekali pakai. Bukan Anda yang meminta? Abaikan email ini.</p>`,
				mail.Button(link, "Masuk sekarang"), code, loginTokenTTLMinutes))
			go s.Mail.Send(email, "Masuk tanpa password - "+s.appName(), body)
		} else {
			logErr("login link %s: %v", email, err)
		}
	}
	s.Redirect(w, r, "/login/link?sent="+email)
}

// consumeLoginToken validasi baris token: aktif, belum kedaluwarsa, belum
// terpakai, attempts < maks. Gagal → (0, "", false) dan baris invalid dihapus
// bila attempts habis.
func (s *Server) consumeLoginToken(id int64) (int64, string, bool) {
	var userID int64
	var email, codeHash string
	var expired, used int
	var attempts int
	err := s.DB.QueryRow(`SELECT user_id, email, code_hash, expires_at <= datetime('now'), used_at IS NOT NULL, attempts
		FROM login_tokens WHERE id=?`, id).
		Scan(&userID, &email, &codeHash, &expired, &used, &attempts)
	if err != nil || expired == 1 || used == 1 || attempts >= loginTokenMaxAttempts {
		return 0, "", false
	}
	return userID, email, codeHash != ""
}

// loginLinkConsume GET /login/link/{token} — klik tautan dari email:
// buat sesi, tandai email verified, hapus token (sekali pakai).
func (s *Server) loginLinkConsume(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.NotFound(w, r)
		return
	}
	var id int64
	var tokenHash string
	err := s.DB.QueryRow(`SELECT id, token_hash FROM login_tokens WHERE token_hash=? AND used_at IS NULL`,
		hashOTP(token)).Scan(&id, &tokenHash)
	if err != nil {
		s.flashErr(w, r, "/login", "Tautan masuk tidak berlaku atau sudah dipakai.")
		return
	}
	userID, email, ok := s.consumeLoginToken(id)
	if !ok || subtle.ConstantTimeCompare([]byte(hashOTP(token)), []byte(tokenHash)) != 1 {
		s.flashErr(w, r, "/login", "Tautan masuk tidak berlaku.")
		return
	}
	var role string
	if err := s.DB.QueryRow(`SELECT role FROM users WHERE id=?`, userID).Scan(&role); err != nil {
		s.flashErr(w, r, "/login", "Akun tidak ditemukan.")
		return
	}
	if _, err := auth.StartSession(s.DB, w, userID, s.cookieCfg()); err != nil {
		jsonError(w, http.StatusInternalServerError, "gagal membuat sesi")
		return
	}
	s.DB.Exec(`UPDATE login_tokens SET used_at=datetime('now') WHERE id=?`, id)
	s.DB.Exec(`UPDATE users SET email_verified_at=COALESCE(email_verified_at, datetime('now')) WHERE id=?`, userID)
	audit(s.DB, "login.link", email, "")
	s.flashOK(w, r, verifiedRedirect(role), "Anda masuk tanpa password via tautan email.")
}

// loginLinkVerify POST /login/link/verify — masuk memakai kode 6 digit.
func (s *Server) loginLinkVerify(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.PostFormValue("email")))
	code := strings.TrimSpace(r.PostFormValue("code"))

	var id, userID int64
	var codeHash string
	var attempts int
	err := s.DB.QueryRow(`SELECT id, user_id, code_hash, attempts FROM login_tokens
		WHERE email=? AND used_at IS NULL AND expires_at > datetime('now') ORDER BY id DESC LIMIT 1`, email).
		Scan(&id, &userID, &codeHash, &attempts)
	if err != nil {
		s.renderWithErrors(w, r, "Auth/MagicLink", inertia.Props{
			"email":   email,
			"sent_to": email,
		}, inertia.ValidationErrors{"code": "Tautan/kode tidak berlaku - minta yang baru."})
		return
	}
	if attempts >= loginTokenMaxAttempts {
		s.DB.Exec(`DELETE FROM login_tokens WHERE id=?`, id)
		s.renderWithErrors(w, r, "Auth/MagicLink", inertia.Props{
			"email": email, "sent_to": email,
		}, inertia.ValidationErrors{"code": "Percobaan melebihi batas - minta kode baru."})
		return
	}
	if subtle.ConstantTimeCompare([]byte(hashOTP(code)), []byte(codeHash)) != 1 {
		s.DB.Exec(`UPDATE login_tokens SET attempts=attempts+1 WHERE id=?`, id)
		errs := inertia.ValidationErrors{"code": fmt.Sprintf("Kode salah (sisa %d percobaan).", loginTokenMaxAttempts-attempts-1)}
		s.renderWithErrors(w, r, "Auth/MagicLink", inertia.Props{"email": email, "sent_to": email}, errs)
		return
	}
	var role string
	if err := s.DB.QueryRow(`SELECT role FROM users WHERE id=?`, userID).Scan(&role); err != nil {
		s.renderWithErrors(w, r, "Auth/MagicLink", inertia.Props{"email": email, "sent_to": email},
			inertia.ValidationErrors{"code": "Akun tidak ditemukan."})
		return
	}
	if _, err := auth.StartSession(s.DB, w, userID, s.cookieCfg()); err != nil {
		jsonError(w, http.StatusInternalServerError, "gagal membuat sesi")
		return
	}
	s.DB.Exec(`UPDATE login_tokens SET used_at=datetime('now') WHERE id=?`, id)
	s.DB.Exec(`UPDATE users SET email_verified_at=COALESCE(email_verified_at, datetime('now')) WHERE id=?`, userID)
	audit(s.DB, "login.kode", email, "")
	s.flashOK(w, r, verifiedRedirect(role), "Anda masuk tanpa password via kode email.")
}

// loginLinkCheck GET /login/link/check?email= — halaman entri kode setelah
// email terkirim (tanpa membocorkan keberadaan akun).
func (s *Server) loginLinkCheck(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := auth.Resolve(s.DB, r); ok {
		s.Redirect(w, r, "/dashboard")
		return
	}
	s.Page(w, r, "Auth/MagicLink", inertia.Props{
		"email":   strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email"))),
		"sent_to": strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email"))),
		"flash":   flashFromQuery(r),
	})
}
