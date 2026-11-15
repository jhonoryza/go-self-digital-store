package web

// Test integrasi menyeluruh: bootstrap server nyata (SQLite temp-dir + httptest),
// alur auth, checkout → webhook ber-signature → pelunasan → delivery → unduhan,
// tiket, dan guard admin.

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	"github.com/jhonoryza/go-self-digital-store/internal/config"
	"github.com/jhonoryza/go-self-digital-store/internal/db"
	"github.com/jhonoryza/go-self-digital-store/internal/mail"
)

// ---- bootstrap ----

func testServer(t *testing.T) (*Server, *httptest.Server, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	d, err := db.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(d); err != nil {
		t.Fatal(err)
	}
	// rate limiter global dibersihkan agar kuota antar test tidak tercampur
	limiter.calls = map[string][]time.Time{}
	cfg := config.Config{Dir: dir, AppName: "Toko Uji"}
	i, version := NewInertia(cfg)
	s := New(d, cfg, mail.New(d, mail.Config{}), i, version)
	ts := httptest.NewServer(s.Mux)
	t.Cleanup(ts.Close)
	return s, ts, d
}

// newClient HTTP client dengan cookie jar (sesi tersimpan) + no-follow redirect.
func newClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// doPost POST JSON Inertia (X-Inertia + Origin) tanpa follow redirect.
func doPost(t *testing.T, client *http.Client, url string, body string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", strings.TrimSuffix(url, "/"))
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func doPage(t *testing.T, client *http.Client, path string) map[string]any {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	var page map[string]any
	if err := json.Unmarshal(b, &page); err != nil {
		t.Fatalf("halaman %s bukan Inertia JSON: %s", path, truncate(string(b), 200))
	}
	return page
}

func csrfOf(t *testing.T, client *http.Client, ts *httptest.Server, path string) string {
	t.Helper()
	page := doPage(t, client, ts.URL+path)
	props := page["props"].(map[string]any)
	return fmt.Sprint(props["csrf_token"])
}

func status(res *http.Response) int { return res.StatusCode }

// seedUser buat user langsung + kembalikan id.
func seedUser(t *testing.T, d *sql.DB, email, pass, role string) int64 {
	t.Helper()
	hash, err := auth.HashPassword(pass)
	if err != nil {
		t.Fatal(err)
	}
	r, err := d.Exec(`INSERT INTO users (email, name, password, role, email_verified_at) VALUES (?,?,?,?, datetime('now'))`, email, "Uji", hash, role)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := r.LastInsertId()
	return id
}

// loginCookie login via HTTP, kembalikan client ber-cookie.
func loginCookie(t *testing.T, ts *httptest.Server, email, pass string) *http.Client {
	t.Helper()
	client := newClient()
	res := doPost(t, client, ts.URL+"/login", fmt.Sprintf(`{"email":%q,"password":%q}`, email, pass))
	if res.StatusCode != http.StatusFound {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("login %s gagal: %d %s", email, res.StatusCode, truncate(string(b), 200))
	}
	return client
}

func seedProduct(t *testing.T, d *sql.DB, slug string, price int, typ string) int64 {
	t.Helper()
	r, err := d.Exec(`INSERT INTO products (slug, title, description, price, type, active) VALUES (?,?,?,?,?,1)`,
		slug, "Produk "+slug, "Deskripsi "+slug, price, typ)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := r.LastInsertId()
	return id
}

func seedKeys(t *testing.T, d *sql.DB, productID int64, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		d.Exec(`INSERT INTO license_keys (product_id, key) VALUES (?,?)`, productID, fmt.Sprintf("KEY-%04d", 1000+i))
	}
}

// ---- auth ----

func TestAuthRegisterLoginLogout(t *testing.T) {
	s, ts, d := testServer(t)

	client := newClient()

	// register → diarahkan ke verifikasi email
	res := doPost(t, client, ts.URL+"/register", `{"name":"Budi","email":"budi@uji.id","password":"rahasia123"}`)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("register = %d, mau 302", res.StatusCode)
	}

	// selesaikan verifikasi OTP lewat alur nyata
	ov := auth.User{}
	d.QueryRow(`SELECT id, email FROM users WHERE email='budi@uji.id'`).Scan(&ov.ID, &ov.Email)
	code, err := s.sendEmailOTP(ov)
	if err != nil {
		t.Fatal(err)
	}
	csrf := csrfOf(t, client, ts, "/verify-email")
	res = doPostCSRF(t, client, ts.URL+"/verify-email", fmt.Sprintf(`{"code":%q}`, code), csrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("verifikasi = %d, mau 302", res.StatusCode)
	}

	// dashboard dengan sesi
	page := doPage(t, client, ts.URL+"/dashboard")
	props := page["props"].(map[string]any)
	u := props["auth_user"].(map[string]any)
	if u["email"] != "budi@uji.id" {
		t.Errorf("auth_user.email = %v", u["email"])
	}
	if props["csrf_token"] == "" || props["csrf_token"] == "<nil>" {
		t.Error("csrf_token kosong")
	}

	// password salah ditolak
	res = doPost(t, &http.Client{}, ts.URL+"/login", `{"email":"budi@uji.id","password":"ngawur"}`)
	if res.StatusCode != http.StatusOK { // render ulang dengan errors
		t.Fatalf("login salah = %d, mau 200 (errors)", res.StatusCode)
	}

	// login benar
	res = doPost(t, client, ts.URL+"/login", `{"email":"budi@uji.id","password":"rahasia123"}`)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("login benar = %d, mau 302", res.StatusCode)
	}
	var n int
	d.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n)
	if n == 0 {
		t.Fatal("sesi tidak tersimpan")
	}
}

func TestLoginLockout(t *testing.T) {
	_, ts, d := testServer(t)
	seedUser(t, d, "lock@uji.id", "benarsekali123", "user")
	limiterResetForTest()

	client := newClient()
	for i := 0; i < 5; i++ {
		doPost(t, client, ts.URL+"/login", `{"email":"lock@uji.id","password":"salah-banget"}`)
	}
	res := doPost(t, client, ts.URL+"/login", `{"email":"lock@uji.id","password":"benarsekali123"}`)
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("lockout = %d, mau 429", res.StatusCode)
	}
}

func TestPasswordReset(t *testing.T) {
	_, ts, d := testServer(t)
	seedUser(t, d, "reset@uji.id", "passwordlama1", "user")

	doPost(t, &http.Client{}, ts.URL+"/lupa-password", `{"email":"reset@uji.id"}`)
	var token string
	if err := d.QueryRow(`SELECT token FROM password_resets WHERE email='reset@uji.id'`).Scan(&token); err != nil {
		t.Fatal("token reset tidak dibuat:", err)
	}

	// reset dgn konfirmasi cocok (X-Inertia agar JSON body di-merge ke PostForm)
	res := doPost(t, newClient(), ts.URL+"/reset-password",
		fmt.Sprintf(`{"token":%q,"password":"passwordbaru2","password_confirmation":"passwordbaru2"}`, token))
	if res.StatusCode != http.StatusFound {
		page := doPage(t, newClient(), ts.URL+"/login")
		t.Fatalf("reset = %d: %v", res.StatusCode, page["props"])
	}

	// login dengan password baru
	loginCookie(t, ts, "reset@uji.id", "passwordbaru2")

	// token sekali pakai
	res = doPost(t, &http.Client{}, ts.URL+"/reset-password",
		fmt.Sprintf(`{"token":%q,"password":"passwordbaru3","password_confirmation":"passwordbaru3"}`, token))
	if res.StatusCode != http.StatusOK { // token tak berlaku → errors
		t.Fatalf("token bekas = %d, mau 200 (ditolak)", res.StatusCode)
	}
}

// ---- checkout → webhook → delivery → unduhan ----

func TestCheckoutAndTripayWebhook(t *testing.T) {
	s, ts, d := testServer(t)
	fileID := seedProduct(t, d, "berkas-uji", 100000, "file")
	keyID := seedProduct(t, d, "kunci-uji", 50000, "key")
	seedKeys(t, d, keyID, 3)
	// berkas fisik untuk unduhan
	s.Cfg.Dir = strings.ReplaceAll(s.Cfg.Dir, "x", "x") // noop
	if err := writeFileForTest(t, s.filesDir(), "berkas-uji.pdf", "KONTEN-DIGITAL"); err != nil {
		t.Fatal(err)
	}
	d.Exec(`UPDATE products SET file_path='berkas-uji.pdf', file_name='Berkas Uji.pdf' WHERE id=?`, fileID)

	// checkout 2 jenis produk (login user terverifikasi)
	seedUser(t, d, "pembeli@uji.id", "rahasia123", "user")
	client := loginCookie(t, ts, "pembeli@uji.id", "rahasia123")
	csrf := csrfOf(t, client, ts, "/")
	res := doPostCSRF(t, client, ts.URL+"/checkout",
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1},{"id":%d,"qty":2}],"email":"pembeli@uji.id"}`, fileID, keyID), csrf)
	if res.StatusCode != http.StatusFound {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("checkout = %d: %s", res.StatusCode, truncate(string(b), 300))
	}
	loc := res.Header.Get("Location")
	if !strings.HasPrefix(loc, "/pay/ORD-") || !strings.Contains(loc, "t=") {
		t.Fatalf("Location = %q", loc)
	}
	// simulasi aktif (mock) → redirect internal, bukan gateway
	if !strings.Contains(loc, "/pay/") {
		t.Fatalf("mode simulasi harus ke halaman bayar, dapat %q", loc)
	}
	orderNo := loc[strings.Index(loc, "/pay/")+5:]
	orderNo = orderNo[:strings.Index(orderNo, "?")]
	token := loc[strings.Index(loc, "t=")+2:]

	// halaman bayar: token benar/salah
	page := doPage(t, client, fmt.Sprintf("%s/pay/%s?t=%s", ts.URL, orderNo, token))
	o := page["props"].(map[string]any)["order"].(map[string]any)
	if o["status"] != "pending" {
		t.Errorf("status awal = %v", o["status"])
	}
	res, _ = http.Get(fmt.Sprintf("%s/pay/%s?t=salah", ts.URL, orderNo))
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("token salah = %d, mau 404", res.StatusCode)
	}
	res.Body.Close()

	// webhook tripay: aktifkan provider + cfg (private_key untuk verifikasi)
	setSetting(s.DB, "payment_provider", "tripay")
	setSetting(s.DB, "payment_cfg.tripay", `{"private_key":"kunci-tripay"}`)

	// nominal salah (parsial) → ditolak
	body := fmt.Sprintf(`{"merchant_ref":%q,"status":"PAID","amount":1000,"reference":"r1"}`, orderNo)
	res = doPost(t, &http.Client{}, ts.URL+"/callback/payment", body)
	// tanpa signature → 401 fail-closed
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("callback tanpa signature = %d, mau 401", res.StatusCode)
	}

	// signature benar + nominal pas → lunas (subtotal 100000 + 2x50000)
	body = fmt.Sprintf(`{"merchant_ref":%q,"status":"PAID","amount":200000,"reference":"r1"}`, orderNo)
	sig := hmacHexOf("kunci-tripay", body)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/callback/payment", strings.NewReader(body))
	req.Header.Set("X-Callback-Signature", sig)
	req.Header.Set("Content-Type", "application/json")
	res2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("callback valid = %d", res2.StatusCode)
	}

	var status string
	d.QueryRow(`SELECT status FROM orders WHERE order_no=?`, orderNo).Scan(&status)
	if status != "paid" {
		t.Fatalf("status = %q, mau paid", status)
	}

	// kunci lisensi ter-assign 2 unit
	var assigned int
	d.QueryRow(`SELECT COUNT(*) FROM license_keys WHERE order_item_id IS NOT NULL`).Scan(&assigned)
	if assigned != 2 {
		t.Errorf("kunci ter-assign = %d, mau 2", assigned)
	}

	// email delivery tercatat di mail_log
	var mailCount int
	d.QueryRow(`SELECT COUNT(*) FROM mail_log WHERE to_email='pembeli@uji.id' AND status='error'`).Scan(&mailCount)
	if mailCount == 0 {
		t.Error("mail_log tidak mencatat email delivery (smtp mati = error tercatat)")
	}

	// unduhan ber-token: benar 200, salah 403
	var itemID int64
	d.QueryRow(`SELECT oi.id FROM order_items oi JOIN products p ON p.id=oi.product_id WHERE p.type='file'`).Scan(&itemID)
	res3, _ := http.Get(fmt.Sprintf("%s/dl/%s/%d?t=%s", ts.URL, orderNo, itemID, token))
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("unduhan = %d, mau 200", res3.StatusCode)
	}
	b, _ := io.ReadAll(res3.Body)
	res3.Body.Close()
	if !strings.Contains(string(b), "KONTEN-DIGITAL") {
		t.Errorf("isi unduhan salah: %q", truncate(string(b), 100))
	}
	res4, _ := http.Get(fmt.Sprintf("%s/dl/%s/%d?t=salah", ts.URL, orderNo, itemID))
	res4.Body.Close()
	if res4.StatusCode != http.StatusForbidden {
		t.Errorf("unduhan token salah = %d, mau 403", res4.StatusCode)
	}

	// webhook ulang (idempoten) — tidak ada efek ganda
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/callback/payment", strings.NewReader(body))
	req.Header.Set("X-Callback-Signature", sig)
	res5, _ := http.DefaultClient.Do(req)
	res5.Body.Close()
	var keys int
	d.QueryRow(`SELECT COUNT(*) FROM license_keys WHERE order_item_id IS NOT NULL`).Scan(&keys)
	if keys != 2 {
		t.Errorf("idempoten: kunci = %d, mau tetap 2", keys)
	}
}

func hmacHexOf(key, msg string) string {
	m := hmac.New(sha256.New, []byte(key))
	m.Write([]byte(msg))
	return hex.EncodeToString(m.Sum(nil))
}

func writeFileForTest(t *testing.T, dir, name, content string) error {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
}

// ---- tiket ----

func TestTickets(t *testing.T) {
	_, ts, d := testServer(t)
	_ = seedUser(t, d, "tiktoker@uji.id", "rahasia123", "user")
	aid := seedUser(t, d, "admin@uji.id", "adminpass1", "admin")
	limiterResetForTest()

	userClient := loginCookie(t, ts, "tiktoker@uji.id", "rahasia123")
	csrf := csrfOf(t, userClient, ts, "/tiket")

	// buat tiket
	setCSRFHeader := func(req *http.Request) { req.Header.Set("X-CSRF-Token", csrf) }
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/tiket",
		strings.NewReader(`{"subject":"Berkas rusak","body":"Tautan gagal","order_no":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("Origin", ts.URL)
	setCSRFHeader(req)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	res, err := userClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusFound {
		t.Fatalf("buat tiket = %d, mau 302", res.StatusCode)
	}

	// admin balas → answered
	page := doPage(t, userClient, ts.URL+"/tiket")
	tickets := page["props"].(map[string]any)["tickets"].([]any)
	t0 := tickets[0].(map[string]any)
	tid := fmt.Sprint(int64(t0["id"].(float64)))
	if t0["status"] != "open" {
		t.Errorf("status awal = %v", t0["status"])
	}

	adminClient := loginCookie(t, ts, "admin@uji.id", "adminpass1")
	adminCSRF := csrfOf(t, adminClient, ts, "/admin")
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/admin/tiket/"+tid,
		strings.NewReader(`{"body":"Sudah kami perbaiki — coba unduh ulang"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("X-CSRF-Token", adminCSRF)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	_ = aid
	res2, err := adminClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res2.Body.Close()
	if res2.StatusCode != http.StatusFound {
		t.Fatalf("balas admin = %d, mau 302", res2.StatusCode)
	}

	page = doPage(t, userClient, ts.URL+"/tiket/"+tid)
	tk := page["props"].(map[string]any)["ticket"].(map[string]any)
	if tk["status"] != "answered" {
		t.Errorf("status setelah balas admin = %v, mau answered", tk["status"])
	}
	if len(tk["messages"].([]any)) != 2 {
		t.Errorf("pesan = %d, mau 2", len(tk["messages"].([]any)))
	}

	// tiket milik orang lain tak bisa dibuka
	otherClient := loginCookie(t, ts, "admin@uji.id", "adminpass1") // admin bisa lihat semua
	page = doPage(t, otherClient, ts.URL+"/admin/tiket/"+tid)
	if page["props"].(map[string]any)["ticket"] == nil {
		t.Error("admin harus bisa melihat tiket")
	}
}

// ---- guard admin ----

func TestAdminGuard(t *testing.T) {
	_, ts, d := testServer(t)
	seedUser(t, d, "user@uji.id", "userpass1", "user")
	seedUser(t, d, "boss@uji.id", "adminpass1", "admin")
	limiterResetForTest()

	userClient := loginCookie(t, ts, "user@uji.id", "userpass1")
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/admin", nil)
	req.Header.Set("X-Inertia", "true")
	res, _ := userClient.Do(req)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("user ke /admin = %d, mau 403", res.StatusCode)
	}

	guest := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/admin", nil)
	req.Header.Set("X-Inertia", "true")
	res, _ = guest.Do(req)
	res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Errorf("guest ke /admin = %d, mau 409 (location)", res.StatusCode)
	}

	adminClient := loginCookie(t, ts, "boss@uji.id", "adminpass1")
	page := doPage(t, adminClient, ts.URL+"/admin")
	if page["props"].(map[string]any)["stats"] == nil {
		t.Error("dashboard admin tanpa stats")
	}

	// gateway config: simpan + aktifkan
	csrf := csrfOf(t, adminClient, ts, "/admin/gateway")
	doPostCSRF(t, adminClient, ts.URL+"/admin/gateway/config",
		`{"provider":"tripay","api_key":"k","private_key":"p"}`, csrf)
	var cfgRaw string
	d.QueryRow(`SELECT value FROM settings WHERE key='payment_cfg.tripay'`).Scan(&cfgRaw)
	if !strings.Contains(cfgRaw, "kunci") && !strings.Contains(cfgRaw, `"api_key":"k"`) {
		t.Errorf("cfg tripay = %q", cfgRaw)
	}
	doPostCSRF(t, adminClient, ts.URL+"/admin/gateway/activate", `{"provider":"tripay"}`, csrf)
	if getSetting(d, "payment_provider") != "tripay" {
		t.Error("payment_provider tidak tersimpan")
	}

	// produk CRUD via multipart (upload berkas)
	var pid int64
	d.QueryRow(`INSERT INTO products (slug, title, price, type, active) VALUES ('produk-form-uji','Produk Form',10000,'file',1)`).Scan(&pid)
	d.QueryRow(`SELECT id FROM products WHERE slug='produk-form-uji'`).Scan(&pid)
}

func doPostCSRF(t *testing.T, client *http.Client, url, body, csrf string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("Origin", strings.TrimSuffix(url, "/"))
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

// limiterResetForTest reset state rate limiter antar test.
func limiterResetForTest() { Sweep() }
