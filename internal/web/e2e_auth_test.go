package web

// Blok test dipecah per domain dari e2e_test.go (helper tetap di sana).

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	"github.com/jhonoryza/go-self-digital-store/internal/config"
)

// ---- auth: validasi + guard sesi + logout ----

func TestE2EAuthValidationAndSession(t *testing.T) {
	s, ts, d := testServer(t)
	seedUser(t, d, "lama@uji.id", "password999", "user")

	// Kuota register 5/menit dihitung wrapper+handler (2 slot per request), jadi
	// tiap percobaan register pakai XFF berbeda (trusted proxy) agar bucket kuota
	// per-IP tidak saling makan; batas per-IP sendiri diuji di SecurityMiddleware.
	s.Cfg.TrustedProxy = true
	regPost := func(client *http.Client, body, xff string) *http.Response {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Inertia", "true")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("Origin", ts.URL)
		req.Header.Set("X-Forwarded-For", xff)
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { res.Body.Close() })
		return res
	}

	// register validasi (email, nama, password)
	res := regPost(&http.Client{}, `{"email":"bukan-email","name":"","password":"pendek"}`, "10.9.9.1")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("register tak valid = %d, mau 200", res.StatusCode)
	}
	errs := postErrors(t, res)
	for _, f := range []string{"email", "name", "password"} {
		if !errHas(errs, f) {
			t.Errorf("errors tanpa %s: %v", f, errs)
		}
	}

	// register sukses → sesi + diarahkan ke verifikasi email
	client := newClient()
	res = regPost(client, `{"name":"Cici","email":"cici@uji.id","password":"rahasia123"}`, "10.9.9.2")
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "/verify-email") {
		t.Fatalf("register = %d %s, mau redirect /verify-email", res.StatusCode, res.Header.Get("Location"))
	}
	// duplikat email ditolak
	res = regPost(&http.Client{}, `{"name":"Lagi","email":"cici@uji.id","password":"rahasia123"}`, "10.9.9.3")
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "email") {
		t.Errorf("duplikat email = %d", res.StatusCode)
	}

	// login email tak dikenal → errors (timing equalizer, tanpa 500)
	res = doPost(t, &http.Client{}, ts.URL+"/login", `{"email":"hantu@uji.id","password":"apa-aja-12"}`)
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "email") {
		t.Errorf("login hantu = %d", res.StatusCode)
	}

	// GET /login saat sudah login → redirect dashboard
	res = doGetRes(t, client, ts.URL+"/login", map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusFound || res.Header.Get("Location") != "/dashboard" {
		t.Errorf("login page saat login = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// verifikasi dulu (dashboard kini butuh email terverifikasi)
	verifyUser(t, s, d, client, ts, "cici@uji.id")
	// logout (perlu CSRF) → sesi dihapus
	csrf := csrfOf(t, client, ts, "/dashboard")
	res = doPostCSRF(t, client, ts.URL+"/logout", `{}`, csrf)
	if res.StatusCode != http.StatusFound || res.Header.Get("Location") != "/" {
		t.Errorf("logout = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	var n int
	d.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n)
	if n != 0 {
		t.Errorf("sesi tersisa = %d, mau 0", n)
	}
	// dashboard setelah logout → 409 + X-Inertia-Location (XHR)
	res = doGetRes(t, client, ts.URL+"/dashboard", map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusConflict || res.Header.Get("X-Inertia-Location") != "/login" {
		t.Errorf("dashboard guest XHR = %d %s", res.StatusCode, res.Header.Get("X-Inertia-Location"))
	}
	// guest non-XHR → 302 /login
	res = doGetRes(t, newClient(), ts.URL+"/dashboard", nil)
	if res.StatusCode != http.StatusFound || res.Header.Get("Location") != "/login" {
		t.Errorf("dashboard guest = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// logout tanpa CSRF → 403, sesi tetap ada
	uid := seedUser(t, d, "dua@uji.id", "password999", "user")
	c2 := loginCookie(t, ts, "dua@uji.id", "password999")
	res = doPost(t, c2, ts.URL+"/logout", `{}`)
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("logout tanpa csrf = %d, mau 403", res.StatusCode)
	}
	d.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id=?`, uid).Scan(&n)
	if n != 1 {
		t.Errorf("sesi user berubah karena csrf gagal: %d", n)
	}
}

// ---- profil + ganti password ----

func TestE2EProfileAndPassword(t *testing.T) {
	s, ts, d := testServer(t)
	seedUser(t, d, "taken@uji.id", "password999", "user")
	uid := seedUser(t, d, "budi@uji.id", "rahasia123", "user")
	client := loginCookie(t, ts, "budi@uji.id", "rahasia123")
	csrf := csrfOf(t, client, ts, "/profil")

	// validasi profil: nama kosong + email salah
	res := doPostCSRF(t, client, ts.URL+"/profil", `{"name":"","email":"ngawur"}`, csrf)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("profil tak valid = %d", res.StatusCode)
	}
	errs := postErrors(t, res)
	if !errHas(errs, "name") || !errHas(errs, "email") {
		t.Errorf("errors profil = %v", errs)
	}
	// email dipakai akun lain
	res = doPostCSRF(t, client, ts.URL+"/profil", `{"name":"Budi","email":"taken@uji.id"}`, csrf)
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "email") {
		t.Errorf("email dipakai = %d", res.StatusCode)
	}

	// update sukses + audit
	res = doPostCSRF(t, client, ts.URL+"/profil", `{"name":"Budi Santoso","email":"budi2@uji.id"}`, csrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("update profil = %d", res.StatusCode)
	}
	var name, email string
	d.QueryRow(`SELECT name, email FROM users WHERE id=?`, uid).Scan(&name, &email)
	if name != "Budi Santoso" || email != "budi2@uji.id" {
		t.Errorf("profil tersimpan = %q %q", name, email)
	}
	var audits int
	d.QueryRow(`SELECT COUNT(*) FROM audit_log WHERE action='profil.update'`).Scan(&audits)
	if audits != 1 {
		t.Errorf("audit profil.update = %d", audits)
	}

	// email diganti → verified direset (temuan audit email-rebind); verifikasi
	// email baru dulu sebelum lanjut (gerbang guardVerified)
	res = doGetRes(t, client, ts.URL+"/dashboard", map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusConflict || res.Header.Get("X-Inertia-Location") != "/verify-email" {
		t.Errorf("dashboard pasca-ganti email = %d %s, mau 409 /verify-email", res.StatusCode, res.Header.Get("X-Inertia-Location"))
	}
	verifyUser(t, s, d, client, ts, "budi2@uji.id")

	// ganti password: current salah
	res = doPostCSRF(t, client, ts.URL+"/profil/password",
		`{"current_password":"salah","password":"baru12345","password_confirmation":"baru12345"}`, csrf)
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "current_password") {
		t.Errorf("current salah = %d", res.StatusCode)
	}
	// konfirmasi tidak cocok
	res = doPostCSRF(t, client, ts.URL+"/profil/password",
		`{"current_password":"rahasia123","password":"baru12345","password_confirmation":"beda12345"}`, csrf)
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "password_confirmation") {
		t.Errorf("konfirmasi beda = %d", res.StatusCode)
	}

	// sesi kedua dibuat sebelum ganti password → putus setelahnya
	client2 := loginCookie(t, ts, "budi2@uji.id", "rahasia123")
	res = doPostCSRF(t, client, ts.URL+"/profil/password",
		`{"current_password":"rahasia123","password":"baru12345","password_confirmation":"baru12345"}`, csrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("ganti password = %d", res.StatusCode)
	}
	res = doGetRes(t, client2, ts.URL+"/dashboard", map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusConflict {
		t.Errorf("sesi lain = %d, mau 409", res.StatusCode)
	}
	res = doGetRes(t, client, ts.URL+"/dashboard", map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusOK {
		t.Errorf("sesi saat ini = %d, mau 200", res.StatusCode)
	}
	// password lama ditolak, baru diterima
	res = doPost(t, &http.Client{}, ts.URL+"/login", `{"email":"budi2@uji.id","password":"rahasia123"}`)
	if res.StatusCode != http.StatusOK {
		t.Errorf("password lama masih diterima = %d", res.StatusCode)
	}
	loginCookie(t, ts, "budi2@uji.id", "baru12345")
}

// ---- middleware keamanan ----

func TestE2ESecurityMiddleware(t *testing.T) {
	s, ts, d := testServer(t)
	seedUser(t, d, "sec@uji.id", "rahasia123", "user")

	// security headers di semua respons
	res := doGetRes(t, newClient(), ts.URL+"/healthz", nil)
	for k, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "same-origin",
	} {
		if got := res.Header.Get(k); got != want {
			t.Errorf("%s = %q, mau %q", k, got, want)
		}
	}

	// origin asing pada method tulis → 403
	res = doPostOrigin(t, &http.Client{}, ts.URL+"/login",
		`{"email":"sec@uji.id","password":"rahasia123"}`, "https://evil.example")
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("origin asing = %d, mau 403", res.StatusCode)
	}
	b, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(b), "origin") {
		t.Errorf("pesan origin = %s", b)
	}

	// CSRF guard: POST ter-guard tanpa header → 403
	client := loginCookie(t, ts, "sec@uji.id", "rahasia123")
	res = doPost(t, client, ts.URL+"/profil", `{"name":"X","email":"sec@uji.id"}`)
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("tanpa csrf = %d, mau 403", res.StatusCode)
	}

	// rate limit register: kuota habis → 429, stabil sampai akhir
	first429 := -1
	for i := 0; i < 6; i++ {
		res = doPost(t, &http.Client{}, ts.URL+"/register", `{"email":"x"}`)
		if res.StatusCode == http.StatusTooManyRequests && first429 < 0 {
			first429 = i + 1
		}
		if first429 > 0 && res.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("429 tidak stabil: req %d = %d", i+1, res.StatusCode)
		}
	}
	if first429 < 0 || first429 > 5 {
		t.Errorf("rate limit tidak terpicu dalam 5 request (pertama di %d)", first429)
	}

	// versi Inertia sesuai JSON halaman (dulu s.Version dibandingkan mentah —
	// gonertia menyimpan md5-nya — sehingga semua GET ber-version selalu 409)
	vpage := doPage(t, newClient(), ts.URL+"/produk")
	servedVersion := fmt.Sprint(vpage["version"])
	res = doGetRes(t, newClient(), ts.URL+"/produk?ok=1", map[string]string{
		"X-Inertia":         "true",
		"X-Inertia-Version": servedVersion,
	})
	if res.StatusCode != http.StatusOK {
		t.Errorf("GET dgn versi benar = %d, mau 200 (versi dilayani %q)", res.StatusCode, servedVersion)
	}

	// versi Inertia salah → 409 + X-Inertia-Location path-relative
	res = doGetRes(t, newClient(), ts.URL+"/produk?ok=1", map[string]string{
		"X-Inertia":         "true",
		"X-Inertia-Version": servedVersion + "x",
	})
	if res.StatusCode != http.StatusConflict || res.Header.Get("X-Inertia-Location") != "/produk?ok=1" {
		t.Errorf("versi beda = %d %s", res.StatusCode, res.Header.Get("X-Inertia-Location"))
	}

	// GET Inertia tanpa versi (fetch shell awal) → dilayani normal
	res = doGetRes(t, newClient(), ts.URL+"/produk", map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusOK {
		t.Errorf("GET tanpa versi = %d, mau 200", res.StatusCode)
	}

	// CORS: aktifkan allowlist, preflight → 204 + header; origin asing tetap 403
	s.Cfg.CorsOrigins = []string{"https://toko.example"}
	req, _ := http.NewRequest(http.MethodOptions, ts.URL+"/login", nil)
	req.Header.Set("Origin", "https://toko.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent || res.Header.Get("Access-Control-Allow-Origin") != "https://toko.example" {
		t.Errorf("preflight = %d ACAO %q", res.StatusCode, res.Header.Get("Access-Control-Allow-Origin"))
	}
	// origin allowlist lolos originCheck (lanjut ke handler: errors login, bukan 403)
	res = doPostOrigin(t, &http.Client{}, ts.URL+"/login",
		`{"email":"hantu@uji.id","password":"salah1234"}`, "https://toko.example")
	if res.StatusCode == http.StatusForbidden {
		t.Error("origin allowlist seharusnya lolos originCheck")
	}
	// origin di luar allowlist ditolak (localhost tak lagi otomatis saat CORS diisi)
	res = doPostOrigin(t, &http.Client{}, ts.URL+"/login",
		`{"email":"hantu@uji.id","password":"salah1234"}`, "https://evil.example")
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("origin asing pasca-CORS = %d, mau 403", res.StatusCode)
	}

	// trusted proxy: XFF dipercaya → IP klien dari header, login tetap berjalan
	s.Cfg.TrustedProxy = true
	limiterResetForTest()
	for i := 0; i < 3; i++ {
		r2 := doPostOrigin(t, &http.Client{}, ts.URL+"/login",
			`{"email":"hantu@uji.id","password":"salah1234"}`, "https://toko.example")
		if r2.StatusCode != http.StatusOK {
			t.Fatalf("login via XFF = %d", r2.StatusCode)
		}
	}
}

// ---- regresi perbaikan audit: email-rebind, otp-cap, dev-login, reset hash ----

func TestE2ESecurityFixes(t *testing.T) {
	s, ts, d := testServer(t, func(c *config.Config) { c.DevLogin = true })
	seedUser(t, d, "boss@uji.id", "adminpass1", "admin")

	// ---- FIX email rebind: ganti email profil → verified direset + OTP baru ----
	client := newClient()
	doPost(t, client, ts.URL+"/register", `{"name":"Rebind","email":"rebind@uji.id","password":"rahasia123"}`)
	verifyUser(t, s, d, client, ts, "rebind@uji.id")
	// pastikan verified: dashboard 200
	res := doGetRes(t, client, ts.URL+"/dashboard", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("dashboard pra-rebind = %d, mau 200", res.StatusCode)
	}
	// ganti email ke alamat lain
	csrf := csrfOf(t, client, ts, "/profil")
	res = doPostCSRF(t, client, ts.URL+"/profil",
		`{"name":"Rebind","email":"emailbaru@uji.id"}`, csrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "/verify-email") {
		t.Fatalf("ganti email = %d %s, mau redirect /verify-email", res.StatusCode, res.Header.Get("Location"))
	}
	// email_verified_at direset → dashboard terkunci (XHR 409 verify-email)
	res = doGetRes(t, client, ts.URL+"/dashboard", map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusConflict || res.Header.Get("X-Inertia-Location") != "/verify-email" {
		t.Errorf("dashboard pasca-ganti email = %d %s, mau 409 /verify-email", res.StatusCode, res.Header.Get("X-Inertia-Location"))
	}
	// OTP baru terkirim ke email baru & verifikasi berhasil (kepemilikan dibuktikan)
	verifyUser(t, s, d, client, ts, "emailbaru@uji.id")
	res = doGetRes(t, client, ts.URL+"/dashboard", nil)
	if res.StatusCode != http.StatusOK {
		t.Errorf("dashboard pasca-verifikasi = %d, mau 200", res.StatusCode)
	}

	// ---- FIX OTP attempt cap: 5 kode salah → OTP dihapus, kode benar pun ditolak ----
	// (user rebind sudah verified — handler langsung redirect; pakai akun baru)
	capC := newClient()
	doPost(t, capC, ts.URL+"/register", `{"name":"Cap","email":"cap@uji.id","password":"rahasia123"}`)
	var cid int64
	d.QueryRow(`SELECT id FROM users WHERE email='cap@uji.id'`).Scan(&cid)
	otp, _ := s.sendEmailOTP(auth.User{ID: cid, Email: "cap@uji.id"})
	cucsr := csrfOf(t, capC, ts, "/verify-email")
	for i := 0; i < 5; i++ {
		doPostCSRF(t, capC, ts.URL+"/verify-email", `{"code":"999999"}`, cucsr)
	}
	res = doPostCSRF(t, capC, ts.URL+"/verify-email", fmt.Sprintf(`{"code":%q}`, otp), cucsr)
	if res.StatusCode != http.StatusOK {
		t.Errorf("kode benar setelah cap = %d, mau 200 (ditolak)", res.StatusCode)
	}
	var otpRow int
	d.QueryRow(`SELECT COUNT(*) FROM email_otps WHERE user_id=?`, cid).Scan(&otpRow)
	if otpRow != 0 {
		t.Errorf("OTP tidak dihapus setelah cap (rows=%d)", otpRow)
	}

	// ---- FIX dev-login: GET /dev/login tanpa flag sudah 404 di server_test;
	// di sini (flag on) pastikan TIDAK mengenumerasi akun ----
	devPage := doGetRes(t, newClient(), ts.URL+"/dev/login", nil)
	dp, _ := io.ReadAll(devPage.Body)
	if strings.Contains(string(dp), "boss@uji.id") || strings.Contains(string(dp), "rebind@uji.id") {
		t.Error("halaman /dev/login masih mengenumerasi email akun")
	}
	// rate limit dev: 21 POST → 429
	limiterResetForTest()
	for i := 0; i < 21; i++ {
		doPost(t, &http.Client{}, ts.URL+"/dev/login", fmt.Sprintf(`{"user_id":%d}`, i))
	}
	if gres := doGetRes(t, newClient(), ts.URL+"/dev/login", nil); gres.StatusCode != http.StatusTooManyRequests {
		t.Errorf("dev login melewati kuota 21/menit")
	}

	// ---- FIX reset token: plaintext TIDAK tersimpan di DB ----
	doPost(t, &http.Client{}, ts.URL+"/lupa-password", `{"email":"emailbaru@uji.id"}`)
	var stored string
	if err := d.QueryRow(`SELECT token FROM password_resets WHERE email='emailbaru@uji.id'`).Scan(&stored); err != nil {
		t.Fatal("token reset tidak dibuat:", err)
	}
	if len(stored) != 64 || stored == "" {
		t.Errorf("token reset = %q, mau hex sha256 64 char", stored)
	}
	// sha256 plaintext dari email tidak bisa ditebak test — verifikasi bentuk;
	// perilaku lookup-by-hash dibuktikan TestPasswordReset
}

// ---- verifikasi email: OTP saat daftar, gerbang dashboard & checkout ----

func TestE2EEmailVerification(t *testing.T) {
	s, ts, d := testServer(t)

	// daftar → redirect ke /verify-email, user belum verified
	client := newClient()
	res := doPost(t, client, ts.URL+"/register", `{"name":"Ovi","email":"ovi@uji.id","password":"rahasia123"}`)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "/verify-email") {
		t.Fatalf("register = %d %s, mau redirect /verify-email", res.StatusCode, res.Header.Get("Location"))
	}
	var verified sql.NullString
	d.QueryRow(`SELECT email_verified_at FROM users WHERE email='ovi@uji.id'`).Scan(&verified)
	if verified.Valid {
		t.Fatal("user baru harus belum verified")
	}

	// dashboard terkunci: XHR 409 + location /verify-email, plain 302
	res = doGetRes(t, client, ts.URL+"/dashboard", map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusConflict || res.Header.Get("X-Inertia-Location") != "/verify-email" {
		t.Errorf("dashboard belum verified = %d %s", res.StatusCode, res.Header.Get("X-Inertia-Location"))
	}
	res = doGetRes(t, client, ts.URL+"/dashboard", nil)
	if res.StatusCode != http.StatusFound || res.Header.Get("Location") != "/verify-email" {
		t.Errorf("dashboard plain = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	// checkout wajib login (guest) → login page
	res = doGetRes(t, newClient(), ts.URL+"/checkout", nil)
	if res.StatusCode != http.StatusFound || res.Header.Get("Location") != "/login" {
		t.Errorf("checkout guest = %d %s, mau ke /login", res.StatusCode, res.Header.Get("Location"))
	}

	// kirim ulang OTP (test ambil kode plaintext via helper resend)
	u := auth.User{ID: 0}
	d.QueryRow(`SELECT id, email FROM users WHERE email='ovi@uji.id'`).Scan(&u.ID, &u.Email)
	u.Email = "ovi@uji.id"
	code, err := s.sendEmailOTP(u)
	if err != nil {
		t.Fatal(err)
	}
	// cooldown: resend kedua langsung ditolak
	res = doPostCSRF(t, client, ts.URL+"/verify-email/resend", `{}`, csrfOf(t, client, ts, "/verify-email"))
	if loc := res.Header.Get("Location"); !strings.Contains(loc, "err=") {
		t.Errorf("resend kedua = %s, mau flash err (cooldown)", loc)
	}

	// kode salah → errors pada respons POST (000000 pasti bukan OTP aktif)
	csrf := csrfOf(t, client, ts, "/verify-email")
	res = doPostCSRF(t, client, ts.URL+"/verify-email", `{"code":"000000"}`, csrf)
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "code") {
		t.Errorf("kode salah = %d, mau 200 dengan errors.code", res.StatusCode)
	}

	// kode benar → verified → dashboard terbuka
	res = doPostCSRF(t, client, ts.URL+"/verify-email", fmt.Sprintf(`{"code":%q}`, code), csrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "/dashboard") {
		t.Fatalf("verify = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	d.QueryRow(`SELECT email_verified_at FROM users WHERE email='ovi@uji.id'`).Scan(&verified)
	if !verified.Valid {
		t.Fatal("email_verified_at tidak terisi")
	}
	dash := doPage(t, client, ts.URL+"/dashboard")
	if _, ok := prop(t, dash)["stats"]; !ok {
		t.Error("dashboard tetap terkunci setelah verifikasi")
	}
	// otp terhapus
	var otps int
	d.QueryRow(`SELECT COUNT(*) FROM email_otps WHERE user_id=?`, u.ID).Scan(&otps)
	if otps != 0 {
		t.Errorf("otp tersisa = %d, mau 0", otps)
	}
}

// ---- login tanpa password: magic link + kode OTP email ----

func TestE2EPasswordlessLogin(t *testing.T) {
	s, ts, d := testServer(t)
	seedUser(t, d, "passless@uji.id", "rahasia123", "user")

	// minta tautan+kode → generik (anti enumerasi), token & kode tersimpan
	res := doPost(t, newClient(), ts.URL+"/login/link", `{"email":"passless@uji.id"}`)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "sent=passless") {
		t.Fatalf("minta tautan = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	var count int
	d.QueryRow(`SELECT COUNT(*) FROM login_tokens WHERE email='passless@uji.id'`).Scan(&count)
	if count != 1 {
		t.Fatalf("token tersimpan = %d, mau 1", count)
	}

	// email tak terdaftar → tetap generik, tidak ada token tersimpan
	res = doPost(t, newClient(), ts.URL+"/login/link", `{"email":"hantu@uji.id"}`)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "sent=hantu") {
		t.Errorf("email hantu = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	d.QueryRow(`SELECT COUNT(*) FROM login_tokens WHERE email='hantu@uji.id'`).Scan(&count)
	if count != 0 {
		t.Error("email tak terdaftar tidak boleh membuat token")
	}

	// METODE 1: magic link → sesi terbentuk, email otomatis verified, token terpakai
	linkToken, _, err := s.createLoginTokens("passless@uji.id")
	if err != nil {
		t.Fatal(err)
	}
	linkClient := newClient()
	res = doGetRes(t, linkClient, ts.URL+"/login/link/"+linkToken, nil)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "/dashboard") {
		t.Fatalf("konsumsi tautan = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	page := doPage(t, linkClient, ts.URL+"/dashboard")
	if _, ok := prop(t, page)["stats"]; !ok {
		t.Error("dashboard tidak terbuka via magic link")
	}
	var verified sql.NullString
	d.QueryRow(`SELECT email_verified_at FROM users WHERE email='passless@uji.id'`).Scan(&verified)
	if !verified.Valid {
		t.Error("magic link harus otomatis memverifikasi email")
	}
	// token sekali pakai
	res = doGetRes(t, newClient(), ts.URL+"/login/link/"+linkToken, nil)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "/login") {
		t.Errorf("tautan bekas = %d %s, mau ditolak", res.StatusCode, res.Header.Get("Location"))
	}

	// METODE 2: kode OTP → sesi terbentuk
	_, code, err := s.createLoginTokens("passless@uji.id")
	if err != nil {
		t.Fatal(err)
	}
	codeClient := newClient()
	res = doPost(t, codeClient, ts.URL+"/login/link/verify",
		fmt.Sprintf(`{"email":"passless@uji.id","code":%q}`, code))
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "/dashboard") {
		t.Fatalf("verifikasi kode = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	page = doPage(t, codeClient, ts.URL+"/dashboard")
	if _, ok := prop(t, page)["stats"]; !ok {
		t.Error("dashboard tidak terbuka via kode")
	}

	// brute force: 5 kode salah → token dihapus
	limiterResetForTest()
	_, code2, _ := s.createLoginTokens("passless@uji.id")
	for i := 0; i < 5; i++ {
		doPost(t, newClient(), ts.URL+"/login/link/verify",
			fmt.Sprintf(`{"email":"passless@uji.id","code":"00000%d"}`, i))
	}
	res = doPost(t, newClient(), ts.URL+"/login/link/verify",
		fmt.Sprintf(`{"email":"passless@uji.id","code":%q}`, code2))
	errs := postErrors(t, res)
	if !errHas(errs, "code") || !strings.Contains(fmt.Sprint(errs["code"]), "minta kode baru") {
		t.Errorf("brute force harus menghapus token: %v", errs)
	}
}
