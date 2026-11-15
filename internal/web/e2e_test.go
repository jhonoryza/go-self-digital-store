package web

// Test E2E menyeluruh per fitur: halaman publik, auth + guard sesi, profil,
// dashboard/pesanan, validasi checkout + kupon, pembayaran simulasi, webhook
// tripay (jalur negatif), unduhan, siklus tiket penuh, konsol admin, dan
// middleware keamanan. Melengkapi server_test.go (alur bahagia utama) — file
// ini menutup cabang validasi, error, dan akses silang semua handler.

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
)

// ---- helper tambahan khusus E2E ----

// errorsOf ambil map errors validasi dari map page Inertia (props.errors;
// gonertia v3 menyimpannya di dalam props).
func errorsOf(t *testing.T, page map[string]any) map[string]any {
	t.Helper()
	props, _ := page["props"].(map[string]any)
	if e, ok := props["errors"].(map[string]any); ok {
		return e
	}
	if e, ok := page["errors"].(map[string]any); ok {
		return e
	}
	t.Fatalf("halaman tanpa errors: %v", page)
	return nil
}

// postErrors baca map errors dari body respons POST Inertia (renderWithErrors).
// Catatan: errors hanya ada pada respons aksi itu sendiri, bukan GET halaman baru.
func postErrors(t *testing.T, res *http.Response) map[string]any {
	t.Helper()
	b, _ := io.ReadAll(res.Body)
	var page map[string]any
	if err := json.Unmarshal(b, &page); err != nil {
		t.Fatalf("respons bukan JSON page: %s", truncate(string(b), 300))
	}
	return errorsOf(t, page)
}

// errHas true bila errors punya field.
func errHas(e map[string]any, field string) bool {
	_, ok := e[field]
	return ok
}

// doGetRes GET mentah dengan header ekstra, tanpa follow redirect.
func doGetRes(t *testing.T, client *http.Client, url string, hdrs map[string]string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

// doPostOrigin POST JSON Inertia dengan Origin eksplisit (uji CORS/origin).
func doPostOrigin(t *testing.T, client *http.Client, url, body, origin string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

// tsOrigin origin dari URL httptest (scheme://host).
func tsOrigin(rawURL string) string {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Scheme + "://" + u.Host
}

// multipartPost POST multipart/form-data Inertia (upload berkas admin).
func multipartPost(t *testing.T, client *http.Client, url, csrf string, fields map[string]string, fileField, fileName, fileContent string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	if fileField != "" {
		fw, _ := w.CreateFormFile(fileField, fileName)
		_, _ = fw.Write([]byte(fileContent))
	}
	_ = w.Close()
	req, _ := http.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", tsOrigin(url))
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

// waitMailLog tunggu mail_log untuk email mencapai jumlah minimum (kirim async).
func waitMailLog(t *testing.T, d *sql.DB, email string, want int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		d.QueryRow(`SELECT COUNT(*) FROM mail_log WHERE to_email=?`, email).Scan(&n)
		if n >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("mail_log %s tidak mencapai %d baris", email, want)
}

// seedCoupon kupon aktif.
func seedCoupon(t *testing.T, d *sql.DB, code string, percent int) {
	t.Helper()
	d.Exec(`INSERT INTO coupons (code, percent, active) VALUES (?,?,1)`, code, percent)
}

// seedOrder order pending langsung di DB + satu item; kembalikan dl_token.
func seedOrder(t *testing.T, d *sql.DB, orderNo, email string, productID int64, price, qty int) string {
	t.Helper()
	token := "tok-" + strings.ToLower(orderNo)
	_, err := d.Exec(`INSERT INTO orders (order_no, email, subtotal, total, dl_token) VALUES (?,?,?,?,?)`,
		orderNo, email, price*qty, price*qty, token)
	if err != nil {
		t.Fatal(err)
	}
	var oid int64
	var title string
	d.QueryRow(`SELECT id FROM orders WHERE order_no=?`, orderNo).Scan(&oid)
	d.QueryRow(`SELECT title FROM products WHERE id=?`, productID).Scan(&title)
	d.Exec(`INSERT INTO order_items (order_id, product_id, title, price, qty) VALUES (?,?,?,?,?)`, oid, productID, title, price, qty)
	return token
}

// parsePayLoc urai Location "/pay/ORD-x?t=token" dari redirect checkout/payNow.
func parsePayLoc(t *testing.T, loc string) (orderNo, token string) {
	t.Helper()
	if !strings.HasPrefix(loc, "/pay/") {
		t.Fatalf("Location bukan halaman bayar: %q", loc)
	}
	rest := loc[len("/pay/"):]
	if qi := strings.Index(rest, "?"); qi >= 0 {
		rest = rest[:qi]
	}
	u, _ := neturl.Parse(loc)
	return rest, u.Query().Get("t")
}

// checkoutOrder POST /checkout sukses; kembalikan order_no + token.
// Checkout kini wajib login terverifikasi → CSRF diambil dari halaman beranda.
func checkoutOrder(t *testing.T, client *http.Client, ts *httptest.Server, body string) (string, string) {
	t.Helper()
	csrf := csrfOf(t, client, ts, "/")
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/checkout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("X-CSRF-Token", csrf)
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	if res.StatusCode != http.StatusFound {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("checkout = %d: %s", res.StatusCode, truncate(string(b), 300))
	}
	return parsePayLoc(t, res.Header.Get("Location"))
}

// verifyUser selesaikan verifikasi email user uji lewat alur OTP nyata
// (kode dari s.sendEmailOTP, POST ke /verify-email).
func verifyUser(t *testing.T, s *Server, d *sql.DB, client *http.Client, ts *httptest.Server, email string) {
	t.Helper()
	u := auth.User{}
	d.QueryRow(`SELECT id, email FROM users WHERE email=?`, email).Scan(&u.ID, &u.Email)
	code, err := s.sendEmailOTP(u)
	if err != nil {
		t.Fatal(err)
	}
	csrf := csrfOf(t, client, ts, "/verify-email")
	res := doPostCSRF(t, client, ts.URL+"/verify-email", fmt.Sprintf(`{"code":%q}`, code), csrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("verify %s = %d", email, res.StatusCode)
	}
}

// paySimulasi POST /pay/{no} mode simulasi (tandai lunas).
func paySimulasi(t *testing.T, client *http.Client, ts *httptest.Server, orderNo, token string) *http.Response {
	t.Helper()
	return doPost(t, client, ts.URL+"/pay/"+orderNo, fmt.Sprintf(`{"t":%q}`, token))
}

// prop ambil props page sebagai map.
func prop(t *testing.T, page map[string]any) map[string]any {
	t.Helper()
	p, ok := page["props"].(map[string]any)
	if !ok {
		t.Fatalf("halaman tanpa props: %v", page)
	}
	return p
}

// ---- halaman publik ----

func TestE2EPublicPages(t *testing.T) {
	_, ts, d := testServer(t)
	pid := seedProduct(t, d, "unggulan-uji", 99000, "file")
	d.Exec(`UPDATE products SET featured=1 WHERE id=?`, pid)
	seedProduct(t, d, "biasa-uji", 10000, "key")
	seedProduct(t, d, "mati-uji", 10000, "file")
	d.Exec(`UPDATE products SET active=0 WHERE slug='mati-uji'`)

	// healthz
	res := doGetRes(t, newClient(), ts.URL+"/healthz", nil)
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(b), `"ok":true`) {
		t.Errorf("healthz = %d %s", res.StatusCode, b)
	}

	// landing: produk unggulan tampil
	page := doPage(t, newClient(), ts.URL+"/")
	feat := prop(t, page)["featured"].([]any)
	if len(feat) != 1 || feat[0].(map[string]any)["slug"] != "unggulan-uji" {
		t.Errorf("featured = %v", feat)
	}

	// shell HTML (tanpa X-Inertia) dirender root template
	res = doGetRes(t, newClient(), ts.URL+"/", nil)
	if res.StatusCode != http.StatusOK {
		t.Errorf("shell = %d", res.StatusCode)
	}
	hb, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(hb), "data-page") {
		t.Error("shell HTML tanpa data-page Inertia")
	}

	// daftar produk hanya aktif
	page = doPage(t, newClient(), ts.URL+"/produk")
	products := prop(t, page)["products"].([]any)
	if len(products) != 2 {
		t.Fatalf("produk = %d, mau 2 (nonaktif disembunyikan)", len(products))
	}
	slugs := map[string]bool{}
	for _, p := range products {
		slugs[p.(map[string]any)["slug"].(string)] = true
	}
	if slugs["mati-uji"] || !slugs["unggulan-uji"] {
		t.Errorf("slug list = %v", slugs)
	}

	// paginasi katalog: tambah 13 produk aktif → total 15 aktif = 2 halaman (12 per halaman)
	for i := 1; i <= 13; i++ {
		seedProduct(t, d, fmt.Sprintf("pag-%02d", i), 1000, "file")
	}
	page = doPage(t, newClient(), ts.URL+"/produk")
	if got := prop(t, page)["last_page"].(float64); got != 2 {
		t.Errorf("last_page = %v, mau 2", got)
	}
	if got := len(prop(t, page)["products"].([]any)); got != 12 {
		t.Errorf("produk halaman 1 = %d, mau 12", got)
	}
	if got := prop(t, page)["total"].(float64); got != 15 {
		t.Errorf("total = %v, mau 15", got)
	}
	// halaman 2 berisi sisa produk
	page = doPage(t, newClient(), ts.URL+"/produk?page=2")
	prods2 := prop(t, page)["products"].([]any)
	if len(prods2) != 3 {
		t.Errorf("produk halaman 2 = %d, mau 3", len(prods2))
	}
	if pg := prop(t, page)["page"].(float64); pg != 2 {
		t.Errorf("page = %v, mau 2", pg)
	}
	// page di luar jangkauan / tidak valid di-clamp, bukan error
	for _, q := range []string{"99", "0", "-1", "bukan-angka"} {
		page = doPage(t, newClient(), ts.URL+"/produk?page="+q)
		if len(prop(t, page)["products"].([]any)) != 12 {
			t.Errorf("page=%s: produk = %d, mau clamp ke halaman 1 (12)", q, len(prop(t, page)["products"].([]any)))
		}
	}

	// detail produk aktif
	page = doPage(t, newClient(), ts.URL+"/produk/unggulan-uji")
	pd := prop(t, page)["product"].(map[string]any)
	if pd["price"].(float64) != 99000 || pd["type"] != "file" {
		t.Errorf("detail = %v", pd)
	}

	// detail nonaktif / tidak ada → 404
	res = doGetRes(t, newClient(), ts.URL+"/produk/mati-uji", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("produk nonaktif = %d, mau 404", res.StatusCode)
	}
	res = doGetRes(t, newClient(), ts.URL+"/produk/tak-ada", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("produk tak ada = %d, mau 404", res.StatusCode)
	}

	// privacy & ketentuan
	for path, kind := range map[string]string{"/privacy": "privacy", "/ketentuan": "terms"} {
		page = doPage(t, newClient(), ts.URL+path)
		pp := prop(t, page)
		if page["component"] != "Public/Policy" || pp["kind"] != kind {
			t.Errorf("%s = component %v kind %v", path, page["component"], pp["kind"])
		}
	}

	// keranjang: halaman wadah (isi keranjang hidup di localStorage klien)
	page = doPage(t, newClient(), ts.URL+"/keranjang")
	if page["component"] != "Public/Cart" {
		t.Errorf("/keranjang = component %v, mau Public/Cart", page["component"])
	}

	// flash query ?ok= / ?err= masuk props
	page = doPage(t, newClient(), ts.URL+"/produk?ok=halo&err=waduh")
	flash := prop(t, page)["flash"].(map[string]any)
	if flash["ok"] != "halo" || flash["err"] != "waduh" {
		t.Errorf("flash = %v", flash)
	}
}

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
	_, ts, d := testServer(t)
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

// ---- dashboard, pesanan, invoice ----

func TestE2EDashboardOrdersInvoice(t *testing.T) {
	s, ts, d := testServer(t)
	fileID := seedProduct(t, d, "berkas-lengkap", 120000, "file")
	if err := writeFileForTest(t, s.filesDir(), "berkas-lengkap.pdf", "DATA"); err != nil {
		t.Fatal(err)
	}
	d.Exec(`UPDATE products SET file_path='berkas-lengkap.pdf', file_name='Berkas Lengkap.pdf' WHERE id=?`, fileID)

	// register + verifikasi email (checkout kini wajib login terverifikasi)
	client := newClient()
	doPost(t, client, ts.URL+"/register", `{"name":"Pembeli","email":"pembeli-lengkap@uji.id","password":"rahasia123"}`)
	verifyUser(t, s, d, client, ts, "pembeli-lengkap@uji.id")

	// dua checkout sebagai user terverifikasi
	ordGuest, tokGuest := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"pembeli-lengkap@uji.id"}`, fileID))

	ordUser, _ := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"pembeli-lengkap@uji.id"}`, fileID))
	var uid sql.NullInt64
	d.QueryRow(`SELECT user_id FROM orders WHERE order_no=?`, ordUser).Scan(&uid)
	if !uid.Valid {
		t.Error("order checkout login tidak terikat user_id")
	}

	// daftar pesanan memuat keduanya
	page := doPage(t, client, ts.URL+"/pesanan")
	orders := prop(t, page)["orders"].([]any)
	if len(orders) != 2 {
		t.Fatalf("orders = %d, mau 2", len(orders))
	}

	// invoice milik sendiri (order guest pun) bisa dibuka
	page = doPage(t, client, ts.URL+"/pesanan/"+ordGuest)
	o := prop(t, page)["order"].(map[string]any)
	if o["order_no"] != ordGuest || o["status"] != "pending" {
		t.Errorf("invoice = %v status %v", o["order_no"], o["status"])
	}
	if len(o["items"].([]any)) != 1 {
		t.Errorf("item invoice = %v", o["items"])
	}

	// order milik orang lain → 404 (user lain harus verified dulu juga)
	doPost(t, newClient(), ts.URL+"/register", `{"name":"Lain","email":"orang-lain@uji.id","password":"rahasia123"}`)
	other := loginCookie(t, ts, "orang-lain@uji.id", "rahasia123")
	verifyUser(t, s, d, other, ts, "orang-lain@uji.id")
	res := doGetRes(t, other, ts.URL+"/pesanan/"+ordGuest, map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("invoice orang lain = %d, mau 404", res.StatusCode)
	}

	// pelunasan simulasi → dashboard stats benar
	res = paySimulasi(t, client, ts, ordGuest, tokGuest)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("paySimulasi = %d", res.StatusCode)
	}
	page = doPage(t, client, ts.URL+"/dashboard")
	stats := prop(t, page)["stats"].(map[string]any)
	if stats["paid_count"].(float64) != 1 || stats["pending_count"].(float64) != 1 {
		t.Errorf("stats hitung = %v", stats)
	}
	if stats["total_spend"].(float64) != 120000 {
		t.Errorf("total_spend = %v", stats["total_spend"])
	}
	if len(stats["recent"].([]any)) != 2 {
		t.Errorf("recent = %d", len(stats["recent"].([]any)))
	}
	// invoice kini status paid
	page = doPage(t, client, ts.URL+"/pesanan/"+ordGuest)
	if prop(t, page)["order"].(map[string]any)["status"] != "paid" {
		t.Error("invoice belum paid")
	}
}

// ---- validasi checkout + kupon ----

func TestE2ECheckoutValidation(t *testing.T) {
	_, ts, d := testServer(t)
	pid := seedProduct(t, d, "produk-checkout", 100000, "key")
	seedKeys(t, d, pid, 1)
	deadID := seedProduct(t, d, "produk-mati", 50000, "file")
	d.Exec(`UPDATE products SET active=0 WHERE id=?`, deadID)

	seedUser(t, d, "validator@uji.id", "rahasia123", "user")
	guest := loginCookie(t, ts, "validator@uji.id", "rahasia123")
	vcsrf := csrfOf(t, guest, ts, "/")

	// keranjang kosong → redirect /keranjang dengan flash err
	res := doPostCSRF(t, guest, ts.URL+"/checkout", `{}`, vcsrf)
	if res.StatusCode != http.StatusFound || !strings.HasPrefix(res.Header.Get("Location"), "/keranjang?") ||
		!strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("keranjang kosong = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// email tidak valid → render ulang dengan errors
	res = doPostCSRF(t, guest, ts.URL+"/checkout", fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"ngawur"}`, pid), vcsrf)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("email salah = %d", res.StatusCode)
	}
	if !errHas(postErrors(t, res), "email") {
		t.Error("errors.email tidak ada")
	}

	// qty di luar 1..10 ditolak
	for _, qty := range []int{0, 11} {
		res = doPostCSRF(t, guest, ts.URL+"/checkout",
			fmt.Sprintf(`{"items":[{"id":%d,"qty":%d}],"email":"p@uji.id"}`, pid, qty), vcsrf)
		if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
			t.Errorf("qty %d = %d %s", qty, res.StatusCode, res.Header.Get("Location"))
		}
	}

	// produk nonaktif ditolak
	res = doPostCSRF(t, guest, ts.URL+"/checkout",
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"p@uji.id"}`, deadID), vcsrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("produk mati = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// kupon valid: diskon 10% (kode lowercase dinormalkan)
	seedCoupon(t, d, "HEMAT10", 10)
	orderNo, _ := checkoutOrder(t, guest, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"p@uji.id","coupon":"hemat10"}`, pid))
	var subtotal, discount, total int
	var coupon string
	d.QueryRow(`SELECT subtotal, discount, total, coupon_code FROM orders WHERE order_no=?`, orderNo).
		Scan(&subtotal, &discount, &total, &coupon)
	if subtotal != 100000 || discount != 10000 || total != 90000 || coupon != "HEMAT10" {
		t.Errorf("kupon: subtotal=%d discount=%d total=%d code=%q", subtotal, discount, total, coupon)
	}

	// kupon tak dikenal → harga penuh
	orderNo, _ = checkoutOrder(t, guest, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"p@uji.id","coupon":"NGASAL"}`, pid))
	d.QueryRow(`SELECT subtotal, discount, total FROM orders WHERE order_no=?`, orderNo).Scan(&subtotal, &discount, &total)
	if discount != 0 || total != 100000 {
		t.Errorf("kupon ngasal: discount=%d total=%d", discount, total)
	}
}

// ---- pembayaran simulasi (mock) ----

func TestE2ESimulatedPayment(t *testing.T) {
	s, ts, d := testServer(t)
	keyID := seedProduct(t, d, "kunci-sim", 25000, "key")
	seedKeys(t, d, keyID, 3)
	noKeyID := seedProduct(t, d, "kunci-habis", 25000, "key") // tanpa stok

	seedUser(t, d, "sim@uji.id", "rahasia123", "user")
	client := loginCookie(t, ts, "sim@uji.id", "rahasia123")
	orderNo, token := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":2}],"email":"sim@uji.id"}`, keyID))

	// halaman bayar: token benar → props simulasi + order pending
	page := doPage(t, client, fmt.Sprintf("%s/pay/%s?t=%s", ts.URL, orderNo, token))
	pp := prop(t, page)
	if pp["simulasi"] != true {
		t.Errorf("simulasi = %v", pp["simulasi"])
	}
	if o := pp["order"].(map[string]any); o["status"] != "pending" || o["total"].(float64) != 50000 {
		t.Errorf("order pay page = %v", o)
	}

	// token salah / kosong / order tak dikenal → 404
	res := doGetRes(t, client, fmt.Sprintf("%s/pay/%s?t=salah", ts.URL, orderNo), nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("token salah = %d", res.StatusCode)
	}
	res = doGetRes(t, client, fmt.Sprintf("%s/pay/%s", ts.URL, orderNo), nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("token kosong = %d", res.StatusCode)
	}
	res = doGetRes(t, client, ts.URL+"/pay/ORD-TAKADA?t=x", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("order tak ada = %d", res.StatusCode)
	}

	// payNow token salah → 404
	res = doPost(t, client, ts.URL+"/pay/"+orderNo, `{"t":"salah"}`)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("payNow token salah = %d", res.StatusCode)
	}

	// payNow benar → lunas, kunci 2 unit, email tercatat
	res = paySimulasi(t, client, ts, orderNo, token)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "ok=") {
		t.Fatalf("payNow = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	var st string
	d.QueryRow(`SELECT status FROM orders WHERE order_no=?`, orderNo).Scan(&st)
	if st != "paid" {
		t.Fatalf("status = %q", st)
	}
	var assigned int
	d.QueryRow(`SELECT COUNT(*) FROM license_keys WHERE order_item_id IS NOT NULL`).Scan(&assigned)
	if assigned != 2 {
		t.Errorf("kunci = %d, mau 2", assigned)
	}
	waitMailLog(t, d, "sim@uji.id", 1)

	// halaman bayar kini menampilkan kunci ter-assign
	page = doPage(t, client, fmt.Sprintf("%s/pay/%s?t=%s", ts.URL, orderNo, token))
	o := prop(t, page)["order"].(map[string]any)
	keys := o["items"].([]any)[0].(map[string]any)["license_keys"].([]any)
	if len(keys) != 2 {
		t.Errorf("kunci di halaman = %d", len(keys))
	}

	// payNow ulang → idempoten, kunci tetap 2
	res = paySimulasi(t, client, ts, orderNo, token)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "ok=") {
		t.Fatalf("payNow ulang = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	d.QueryRow(`SELECT COUNT(*) FROM license_keys WHERE order_item_id IS NOT NULL`).Scan(&assigned)
	if assigned != 2 {
		t.Errorf("kunci setelah ulang = %d", assigned)
	}

	// order kunci tanpa stok: tetap lunas, 0 kunci, email tetap dikirim
	orderNo2, token2 := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"habis@uji.id"}`, noKeyID))
	res = paySimulasi(t, client, ts, orderNo2, token2)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("payNow tanpa stok = %d", res.StatusCode)
	}
	d.QueryRow(`SELECT status FROM orders WHERE order_no=?`, orderNo2).Scan(&st)
	if st != "paid" {
		t.Errorf("status tanpa stok = %q", st)
	}
	waitMailLog(t, d, "habis@uji.id", 1)

	// webhook saat mode simulasi → 404 (bukan jalur callback)
	res = doPost(t, &http.Client{}, ts.URL+"/callback/payment", `{"order_id":"x","status":"paid","amount":1}`)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("callback simulasi = %d, mau 404", res.StatusCode)
	}

	// payNow saat provider non-simulasi aktif → 404
	setSetting(s.DB, "payment_provider", "tripay")
	setSetting(s.DB, "payment_cfg.tripay", `{"private_key":"k"}`)
	res = doPost(t, client, ts.URL+"/pay/"+orderNo2, fmt.Sprintf(`{"t":%q}`, token2))
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("payNow non-simulasi = %d, mau 404", res.StatusCode)
	}
}

// ---- webhook tripay: jalur negatif ----

func TestE2ETripayWebhookNegatives(t *testing.T) {
	s, ts, d := testServer(t)
	fileID := seedProduct(t, d, "berkas-webhook", 200000, "file")
	seedUser(t, d, "hook@uji.id", "rahasia123", "user")
	hookClient := loginCookie(t, ts, "hook@uji.id", "rahasia123")
	orderNo, _ := checkoutOrder(t, hookClient, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"hook@uji.id"}`, fileID))

	setSetting(s.DB, "payment_provider", "tripay")
	setSetting(s.DB, "payment_cfg.tripay", `{"private_key":"kunci-tripay"}`)

	sign := func(body string) http.Header {
		h := http.Header{}
		h.Set("X-Callback-Signature", hmacHexOf("kunci-tripay", body))
		h.Set("Content-Type", "application/json")
		return h
	}
	sendHook := func(body string, hdr http.Header) (int, string) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/callback/payment", strings.NewReader(body))
		req.Header = hdr
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b)
	}

	// status bukan PAID → diabaikan, order tetap pending
	body := fmt.Sprintf(`{"merchant_ref":%q,"status":"EXPIRED","amount":200000,"reference":"r1"}`, orderNo)
	code, respBody := sendHook(body, sign(body))
	if code != http.StatusOK || !strings.Contains(respBody, `"ignored":true`) {
		t.Errorf("EXPIRED = %d %s", code, respBody)
	}
	var st string
	d.QueryRow(`SELECT status FROM orders WHERE order_no=?`, orderNo).Scan(&st)
	if st != "pending" {
		t.Errorf("status setelah EXPIRED = %q", st)
	}

	// nominal tidak cocok → tidak lunas walau status PAID
	body = fmt.Sprintf(`{"merchant_ref":%q,"status":"PAID","amount":199999,"reference":"r2"}`, orderNo)
	code, respBody = sendHook(body, sign(body))
	if code != http.StatusOK || !strings.Contains(respBody, `"ok":true`) {
		t.Errorf("nominal beda = %d %s", code, respBody)
	}
	d.QueryRow(`SELECT status FROM orders WHERE order_no=?`, orderNo).Scan(&st)
	if st != "pending" {
		t.Errorf("status setelah nominal beda = %q", st)
	}

	// order tak dikenal → ok tanpa efek
	body = `{"merchant_ref":"ORD-TAKADA","status":"PAID","amount":1000,"reference":"r3"}`
	code, _ = sendHook(body, sign(body))
	if code != http.StatusOK {
		t.Errorf("order tak dikenal = %d", code)
	}

	// signature salah → 401
	body = fmt.Sprintf(`{"merchant_ref":%q,"status":"PAID","amount":200000,"reference":"r4"}`, orderNo)
	bad := sign(body)
	bad.Set("X-Callback-Signature", "deadbeef")
	code, _ = sendHook(body, bad)
	if code != http.StatusUnauthorized {
		t.Errorf("signature salah = %d, mau 401", code)
	}

	// fail-closed: secret kosong → 401 walau signature "benar"
	setSetting(s.DB, "payment_cfg.tripay", `{}`)
	code, _ = sendHook(body, sign(body))
	if code != http.StatusUnauthorized {
		t.Errorf("cfg kosong = %d, mau 401", code)
	}

	// method salah → 405
	res := doGetRes(t, newClient(), ts.URL+"/callback/payment", nil)
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET callback = %d, mau 405", res.StatusCode)
	}
}

// ---- unduhan /dl ----

func TestE2EDownloadGuards(t *testing.T) {
	s, ts, d := testServer(t)
	fileID := seedProduct(t, d, "berkas-dl", 50000, "file")
	keyID := seedProduct(t, d, "kunci-dl", 50000, "key")
	seedKeys(t, d, keyID, 1)
	if err := writeFileForTest(t, s.filesDir(), "berkas-dl.zip", "ZIP-DATA-RAHASIA"); err != nil {
		t.Fatal(err)
	}
	d.Exec(`UPDATE products SET file_path='berkas-dl.zip', file_name='Berkas DL.zip' WHERE id=?`, fileID)

	seedUser(t, d, "dl@uji.id", "rahasia123", "user")
	client := loginCookie(t, ts, "dl@uji.id", "rahasia123")
	orderNo, token := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1},{"id":%d,"qty":1}],"email":"dl@uji.id"}`, fileID, keyID))

	var fileItem, keyItem int64
	d.QueryRow(`SELECT oi.id FROM order_items oi JOIN products p ON p.id=oi.product_id WHERE p.type='file'`).Scan(&fileItem)
	d.QueryRow(`SELECT oi.id FROM order_items oi JOIN products p ON p.id=oi.product_id WHERE p.type='key'`).Scan(&keyItem)

	dl := func(item int64, tok string) *http.Response {
		return doGetRes(t, &http.Client{}, fmt.Sprintf("%s/dl/%s/%d?t=%s", ts.URL, orderNo, item, tok), nil)
	}

	// sebelum lunas → 403
	if res := dl(fileItem, token); res.StatusCode != http.StatusForbidden {
		t.Errorf("belum lunas = %d, mau 403", res.StatusCode)
	}

	// lunas via simulasi
	if res := paySimulasi(t, client, ts, orderNo, token); res.StatusCode != http.StatusFound {
		t.Fatalf("payNow = %d", res.StatusCode)
	}

	// token salah / kosong → 403
	if res := dl(fileItem, "salah"); res.StatusCode != http.StatusForbidden {
		t.Errorf("token salah = %d", res.StatusCode)
	}
	res := doGetRes(t, &http.Client{}, fmt.Sprintf("%s/dl/%s/%d", ts.URL, orderNo, fileItem), nil)
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("token kosong = %d", res.StatusCode)
	}

	// item bukan type=file / item tak ada → 404
	if res := dl(keyItem, token); res.StatusCode != http.StatusNotFound {
		t.Errorf("item kunci = %d, mau 404", res.StatusCode)
	}
	if res := dl(99999, token); res.StatusCode != http.StatusNotFound {
		t.Errorf("item ngawur = %d, mau 404", res.StatusCode)
	}

	// unduhan sah → 200 + nama berkas + isi
	res = dl(fileItem, token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("unduhan = %d", res.StatusCode)
	}
	if cd := res.Header.Get("Content-Disposition"); !strings.Contains(cd, "Berkas DL.zip") {
		t.Errorf("disposition = %q", cd)
	}
	b, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(b), "ZIP-DATA-RAHASIA") {
		t.Errorf("isi = %q", truncate(string(b), 100))
	}
}

// ---- tiket: siklus penuh ----

func TestE2ETicketsLifecycle(t *testing.T) {
	_, ts, d := testServer(t)
	uid := seedUser(t, d, "op@uji.id", "rahasia123", "user")
	seedUser(t, d, "lain@uji.id", "rahasia123", "user")
	seedUser(t, d, "admin@uji.id", "adminpass1", "admin")
	pid := seedProduct(t, d, "berkas-tiket", 10000, "file")
	seedOrder(t, d, "ORD-TIKET", "op@uji.id", pid, 10000, 1)
	seedOrder(t, d, "ORD-ORANG", "lain@uji.id", pid, 10000, 1)

	user := loginCookie(t, ts, "op@uji.id", "rahasia123")
	csrf := csrfOf(t, user, ts, "/tiket")

	// validasi: subjek & pesan wajib
	res := doPostCSRF(t, user, ts.URL+"/tiket", `{"subject":"","body":""}`, csrf)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("tiket kosong = %d", res.StatusCode)
	}
	errs := postErrors(t, res)
	if !errHas(errs, "subject") || !errHas(errs, "body") {
		t.Errorf("errors tiket = %v", errs)
	}

	// order milik orang lain ditolak
	res = doPostCSRF(t, user, ts.URL+"/tiket", `{"subject":"S","body":"B","order_no":"ORD-ORANG"}`, csrf)
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "order_no") {
		t.Errorf("order orang lain = %d", res.StatusCode)
	}

	// buat tiket terkait order sendiri
	res = doPostCSRF(t, user, ts.URL+"/tiket",
		`{"subject":"Berkas rusak","body":"Tautan gagal dibuka","order_no":"ORD-TIKET"}`, csrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("buat tiket = %d", res.StatusCode)
	}
	var tid int64
	var orderNo, status string
	d.QueryRow(`SELECT id, order_no, status FROM tickets WHERE user_id=?`, uid).Scan(&tid, &orderNo, &status)
	if orderNo != "ORD-TIKET" || status != "open" {
		t.Errorf("tiket = id %d order %q status %q", tid, orderNo, status)
	}

	// user lain tak bisa membuka tiket itu
	other := loginCookie(t, ts, "lain@uji.id", "rahasia123")
	res = doGetRes(t, other, fmt.Sprintf("%s/tiket/%d", ts.URL, tid), map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("tiket orang lain = %d, mau 404", res.StatusCode)
	}

	// admin balas → answered
	admin := loginCookie(t, ts, "admin@uji.id", "adminpass1")
	acsrf := csrfOf(t, admin, ts, "/admin")
	res = doPostCSRF(t, admin, fmt.Sprintf("%s/admin/tiket/%d", ts.URL, tid), `{"body":"Sudah diperbaiki"}`, acsrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("balas admin = %d", res.StatusCode)
	}
	d.QueryRow(`SELECT status FROM tickets WHERE id=?`, tid).Scan(&status)
	if status != "answered" {
		t.Errorf("status = %q, mau answered", status)
	}

	// balasan user → kembali open
	res = doPostCSRF(t, user, fmt.Sprintf("%s/tiket/%d", ts.URL, tid), `{"body":"Masih gagal"}`, csrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("balas user = %d", res.StatusCode)
	}
	d.QueryRow(`SELECT status FROM tickets WHERE id=?`, tid).Scan(&status)
	if status != "open" {
		t.Errorf("status = %q, mau open", status)
	}
	var msgs int
	d.QueryRow(`SELECT COUNT(*) FROM ticket_messages WHERE ticket_id=?`, tid).Scan(&msgs)
	if msgs != 3 {
		t.Errorf("pesan = %d, mau 3", msgs)
	}

	// admin tutup → closed
	res = doPostCSRF(t, admin, fmt.Sprintf("%s/admin/tiket/%d/close", ts.URL, tid), `{}`, acsrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("tutup tiket = %d", res.StatusCode)
	}
	d.QueryRow(`SELECT status FROM tickets WHERE id=?`, tid).Scan(&status)
	if status != "closed" {
		t.Errorf("status = %q, mau closed", status)
	}

	// balas tiket tertutup → flash err, tanpa pesan baru
	res = doPostCSRF(t, user, fmt.Sprintf("%s/tiket/%d", ts.URL, tid), `{"body":"Lagi"}`, csrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("balas closed = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	d.QueryRow(`SELECT COUNT(*) FROM ticket_messages WHERE ticket_id=?`, tid).Scan(&msgs)
	if msgs != 3 {
		t.Errorf("pesan setelah closed = %d, mau tetap 3", msgs)
	}

	// daftar admin memuat email pemilik + detail pesan
	page := doPage(t, admin, ts.URL+"/admin/tiket")
	found := false
	for _, it := range prop(t, page)["tickets"].([]any) {
		if m := it.(map[string]any); int64(m["id"].(float64)) == tid && m["email"] == "op@uji.id" {
			found = true
		}
	}
	if !found {
		t.Error("tiket tidak muncul di daftar admin")
	}
	page = doPage(t, admin, fmt.Sprintf("%s/admin/tiket/%d", ts.URL, tid))
	tk := prop(t, page)["ticket"].(map[string]any)
	if len(tk["messages"].([]any)) != 3 {
		t.Errorf("pesan detail admin = %d", len(tk["messages"].([]any)))
	}

	// balasan admin tanpa pesan → errors
	res = doPostCSRF(t, admin, fmt.Sprintf("%s/admin/tiket/%d", ts.URL, tid), `{"body":""}`, acsrf)
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "body") {
		t.Errorf("balas kosong = %d", res.StatusCode)
	}
}

// ---- konsol admin: produk, stok kunci, order, gateway ----

func TestE2EAdminConsole(t *testing.T) {
	s, ts, d := testServer(t)
	seedUser(t, d, "user-polos@uji.id", "rahasia123", "user")
	seedUser(t, d, "boss@uji.id", "adminpass1", "admin")
	admin := loginCookie(t, ts, "boss@uji.id", "adminpass1")
	acsrf := csrfOf(t, admin, ts, "/admin")

	// semua halaman admin dirender dengan props utama
	pages := map[string]string{
		"/admin/users":   "users",
		"/admin/orders":  "orders",
		"/admin/produk":  "products",
		"/admin/maillog": "logs",
		"/admin/tiket":   "tickets",
		"/admin/gateway": "providers",
	}
	for path, key := range pages {
		page := doPage(t, admin, ts.URL+path)
		if _, ok := prop(t, page)[key]; !ok {
			t.Errorf("%s tanpa props %s", path, key)
		}
	}

	// create produk key (tanpa berkas) → 302
	res := multipartPost(t, admin, ts.URL+"/admin/produk", acsrf, map[string]string{
		"slug": "kunci-admin", "title": "Kunci Admin", "price": "15000", "type": "key", "active": "1",
	}, "", "", "")
	if res.StatusCode != http.StatusFound {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("create produk = %d: %s", res.StatusCode, truncate(string(b), 300))
	}
	var keyProdID int64
	d.QueryRow(`SELECT id FROM products WHERE slug='kunci-admin'`).Scan(&keyProdID)

	// create type=file tanpa berkas → errors
	res = multipartPost(t, admin, ts.URL+"/admin/produk", acsrf, map[string]string{
		"slug": "file-kosong", "title": "F", "price": "1000", "type": "file",
	}, "", "", "")
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "file") {
		t.Fatalf("file tanpa upload = %d", res.StatusCode)
	}

	// create type=link tanpa ext_url → errors; dengan ext_url → sukses
	res = multipartPost(t, admin, ts.URL+"/admin/produk", acsrf, map[string]string{
		"slug": "link-tanpa-url", "title": "L", "price": "1000", "type": "link",
	}, "", "", "")
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "ext_url") {
		t.Fatalf("link tanpa url = %d", res.StatusCode)
	}
	res = multipartPost(t, admin, ts.URL+"/admin/produk", acsrf, map[string]string{
		"slug": "link-uji", "title": "Link Uji", "price": "20000", "type": "link", "ext_url": "https://contoh.id/unduh",
	}, "", "", "")
	if res.StatusCode != http.StatusFound {
		t.Fatalf("create link = %d", res.StatusCode)
	}
	var extURL string
	d.QueryRow(`SELECT ext_url FROM products WHERE slug='link-uji'`).Scan(&extURL)
	if extURL != "https://contoh.id/unduh" {
		t.Errorf("ext_url = %q", extURL)
	}

	// slug duplikat → errors
	res = multipartPost(t, admin, ts.URL+"/admin/produk", acsrf, map[string]string{
		"slug": "kunci-admin", "title": "Lagi", "price": "15000", "type": "key",
	}, "", "", "")
	if res.StatusCode != http.StatusOK || !errHas(postErrors(t, res), "slug") {
		t.Errorf("dup slug = %d", res.StatusCode)
	}

	// create produk file dengan upload → berkas tersimpan di files/
	res = multipartPost(t, admin, ts.URL+"/admin/produk", acsrf, map[string]string{
		"slug": "file-upload", "title": "Berkas Upload", "price": "30000", "type": "file", "active": "1",
	}, "file", "Panduan.pdf", "ISI-PANDUAN")
	if res.StatusCode != http.StatusFound {
		t.Fatalf("upload produk = %d", res.StatusCode)
	}
	var filePath string
	d.QueryRow(`SELECT file_path FROM products WHERE slug='file-upload'`).Scan(&filePath)
	if filePath == "" {
		t.Fatal("file_path kosong")
	}
	if _, err := os.Stat(filepath.Join(s.Cfg.Dir, "files", filePath)); err != nil {
		t.Errorf("berkas tidak tersimpan: %v", err)
	}

	// update produk: judul + harga berubah
	res = multipartPost(t, admin, fmt.Sprintf("%s/admin/produk/%d", ts.URL, keyProdID), acsrf, map[string]string{
		"slug": "kunci-admin", "title": "Kunci Admin v2", "price": "18000", "type": "key", "active": "1",
	}, "", "", "")
	if res.StatusCode != http.StatusFound {
		t.Fatalf("update produk = %d", res.StatusCode)
	}
	var title string
	var price int
	d.QueryRow(`SELECT title, price FROM products WHERE id=?`, keyProdID).Scan(&title, &price)
	if title != "Kunci Admin v2" || price != 18000 {
		t.Errorf("update = %q %d", title, price)
	}

	// stok kunci massal (multiline)
	res = doPostCSRF(t, admin, fmt.Sprintf("%s/admin/produk/%d/keys", ts.URL, keyProdID),
		`{"keys":"K-1\nK-2\nK-3"}`, acsrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("tambah kunci = %d", res.StatusCode)
	}
	var keys int
	d.QueryRow(`SELECT COUNT(*) FROM license_keys WHERE product_id=?`, keyProdID).Scan(&keys)
	if keys != 3 {
		t.Errorf("kunci = %d, mau 3", keys)
	}
	// key_count di props admin
	page := doPage(t, admin, ts.URL+"/admin/produk")
	for _, it := range prop(t, page)["products"].([]any) {
		if m := it.(map[string]any); m["slug"] == "kunci-admin" && int(m["key_count"].(float64)) != 3 {
			t.Errorf("key_count = %v", m["key_count"])
		}
	}

	// toggle nonaktif → detail publik 404, checkout menolak
	res = doPostCSRF(t, admin, fmt.Sprintf("%s/admin/produk/%d/toggle", ts.URL, keyProdID), `{}`, acsrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("toggle = %d", res.StatusCode)
	}
	res = doGetRes(t, newClient(), ts.URL+"/produk/kunci-admin", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("detail nonaktif = %d, mau 404", res.StatusCode)
	}
	seedUser(t, d, "cart@uji.id", "rahasia123", "user")
	cartClient := loginCookie(t, ts, "cart@uji.id", "rahasia123")
	res = doPostCSRF(t, cartClient, ts.URL+"/checkout",
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"cart@uji.id"}`, keyProdID), csrfOf(t, cartClient, ts, "/"))
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("checkout nonaktif = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	// toggle kembali aktif
	doPostCSRF(t, admin, fmt.Sprintf("%s/admin/produk/%d/toggle", ts.URL, keyProdID), `{}`, acsrf)

	// order manual lunas: kunci ter-assign + audit
	pid := seedProduct(t, d, "produk-manual", 10000, "key")
	seedKeys(t, d, pid, 2)
	seedOrder(t, d, "ORD-MANUAL", "manual@uji.id", pid, 10000, 1)
	res = doPostCSRF(t, admin, ts.URL+"/admin/orders/ORD-MANUAL/paid", `{}`, acsrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "ok=") {
		t.Fatalf("manual paid = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	var status string
	d.QueryRow(`SELECT status FROM orders WHERE order_no='ORD-MANUAL'`).Scan(&status)
	if status != "paid" {
		t.Fatalf("status = %q", status)
	}
	var assigned int
	d.QueryRow(`SELECT COUNT(*) FROM license_keys WHERE order_item_id IS NOT NULL`).Scan(&assigned)
	if assigned != 1 {
		t.Errorf("kunci manual = %d, mau 1", assigned)
	}
	var audits int
	d.QueryRow(`SELECT COUNT(*) FROM audit_log WHERE action='order.paid'`).Scan(&audits)
	if audits != 1 {
		t.Errorf("audit = %d", audits)
	}

	// manual paid ulang → flash err
	res = doPostCSRF(t, admin, ts.URL+"/admin/orders/ORD-MANUAL/paid", `{}`, acsrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("manual paid ulang = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// resend: pending ditolak; paid → email ulang
	seedOrder(t, d, "ORD-PENDING", "pending@uji.id", pid, 10000, 1)
	res = doPostCSRF(t, admin, ts.URL+"/admin/orders/ORD-PENDING/resend", `{}`, acsrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("resend pending = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	res = doPostCSRF(t, admin, ts.URL+"/admin/orders/ORD-MANUAL/resend", `{}`, acsrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "ok=") {
		t.Errorf("resend paid = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	waitMailLog(t, d, "manual@uji.id", 1)

	// gateway: provider tak dikenal → flash err
	res = doPostCSRF(t, admin, ts.URL+"/admin/gateway/config", `{"provider":"gak-ada"}`, acsrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("gateway ngawur = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// config tripay: simpan ulang dengan ••• mempertahankan secret
	doPostCSRF(t, admin, ts.URL+"/admin/gateway/config",
		`{"provider":"tripay","private_key":"kunci-asli","api_key":"lama"}`, acsrf)
	doPostCSRF(t, admin, ts.URL+"/admin/gateway/config",
		`{"provider":"tripay","private_key":"•••","api_key":"baru"}`, acsrf)
	var cfgRaw string
	d.QueryRow(`SELECT value FROM settings WHERE key='payment_cfg.tripay'`).Scan(&cfgRaw)
	if !strings.Contains(cfgRaw, `"private_key":"kunci-asli"`) || !strings.Contains(cfgRaw, `"api_key":"baru"`) {
		t.Errorf("cfg tripay = %q", cfgRaw)
	}

	// halaman gateway: mock simulasi + secret termasked
	page = doPage(t, admin, ts.URL+"/admin/gateway")
	var tripay, mock map[string]any
	for _, it := range prop(t, page)["providers"].([]any) {
		switch m := it.(map[string]any); m["name"] {
		case "tripay":
			tripay = m
		case "mock":
			mock = m
		}
	}
	if mock == nil || mock["simulated"] != true {
		t.Error("mock tidak terdaftar sebagai simulasi")
	}
	if tripay == nil {
		t.Fatal("tripay tidak di daftar")
	}
	masked := false
	for _, f := range tripay["fields"].([]any) {
		m := f.(map[string]any)
		if m["key"] == "private_key" && m["secret"] == true && m["value"] == "•••" {
			masked = true
		}
	}
	if !masked {
		t.Errorf("secret tidak termasked: %v", tripay["fields"])
	}

	// activate tripay
	res = doPostCSRF(t, admin, ts.URL+"/admin/gateway/activate", `{"provider":"tripay"}`, acsrf)
	if res.StatusCode != http.StatusFound || getSetting(d, "payment_provider") != "tripay" {
		t.Errorf("activate = %d %q", res.StatusCode, getSetting(d, "payment_provider"))
	}
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

// ---- unit: format rupiah (bug lama: 59000 → "Rp59.00.0") ----

func TestE2ERupiahFormat(t *testing.T) {
	cases := map[int]string{
		0:         "Rp0",
		5:         "Rp5",
		100:       "Rp100",
		1000:      "Rp1.000",
		59000:     "Rp59.000",
		1250000:   "Rp1.250.000",
		123456789: "Rp123.456.789",
		-99000:    "-Rp99.000",
	}
	for in, want := range cases {
		if got := rupiah(in); got != want {
			t.Errorf("rupiah(%d) = %q, mau %q", in, got, want)
		}
	}
}

// ---- alur lintas fitur: beli → tiket → invoice (satu sesi user) ----

func TestE2EBuyThenTicketThenInvoice(t *testing.T) {
	s, ts, d := testServer(t)
	keyID := seedProduct(t, d, "kunci-lintas", 75000, "key")
	seedKeys(t, d, keyID, 1)

	client := newClient()
	doPost(t, client, ts.URL+"/register", `{"name":"Rina","email":"rina@uji.id","password":"rahasia123"}`)
	verifyUser(t, s, d, client, ts, "rina@uji.id")
	orderNo, token := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"rina@uji.id"}`, keyID))

	// tiket merujuk order yang baru dibuat
	csrf := csrfOf(t, client, ts, "/tiket")
	res := doPostCSRF(t, client, ts.URL+"/tiket",
		fmt.Sprintf(`{"subject":"Tanya kunci","body":"Kapan dikirim?","order_no":%q}`, orderNo), csrf)
	if res.StatusCode != http.StatusFound {
		t.Fatalf("tiket dgn order = %d", res.StatusCode)
	}

	// bayar → invoice menampilkan kunci; dashboard terhitung
	if res := paySimulasi(t, client, ts, orderNo, token); res.StatusCode != http.StatusFound {
		t.Fatalf("payNow = %d", res.StatusCode)
	}
	page := doPage(t, client, ts.URL+"/pesanan/"+orderNo)
	o := prop(t, page)["order"].(map[string]any)
	item := o["items"].([]any)[0].(map[string]any)
	ks := item["license_keys"].([]any)
	if len(ks) != 1 {
		t.Fatalf("kunci invoice = %v", ks)
	}
	var keyVal string
	d.QueryRow(`SELECT key FROM license_keys WHERE order_item_id IS NOT NULL`).Scan(&keyVal)
	if ks[0] != keyVal {
		t.Errorf("kunci invoice %v ≠ DB %q", ks[0], keyVal)
	}
	page = doPage(t, client, ts.URL+"/dashboard")
	st := prop(t, page)["stats"].(map[string]any)
	if st["paid_count"].(float64) != 1 || st["total_spend"].(float64) != 75000 {
		t.Errorf("stats = %v", st)
	}
}

// ---- tautan unduh di invoice/pay: satu tautan per item, multi item didukung ----

func TestE2EMultipleDownloadLinks(t *testing.T) {
	s, ts, d := testServer(t)
	// dua produk file dengan berkas nyata + satu file produk TANPA berkas
	f1 := seedProduct(t, d, "berkas-multi-1", 50000, "file")
	f2 := seedProduct(t, d, "berkas-multi-2", 75000, "file")
	fEmpty := seedProduct(t, d, "berkas-kosong", 10000, "file")
	for _, id := range []int64{f1, f2} {
		if err := writeFileForTest(t, s.filesDir(), fmt.Sprintf("berkas-%d.pdf", id), fmt.Sprintf("KONTEN-%d", id)); err != nil {
			t.Fatal(err)
		}
		d.Exec(`UPDATE products SET file_path=?, file_name=? WHERE id=?`, fmt.Sprintf("berkas-%d.pdf", id), fmt.Sprintf("Berkas %d.pdf", id), id)
	}

	// user checkout 3 produk file sekaligus (2 berkas + 1 kosong) dalam SATU order
	client := newClient()
	doPost(t, client, ts.URL+"/register", `{"name":"Multi","email":"multi@uji.id","password":"rahasia123"}`)
	verifyUser(t, s, d, client, ts, "multi@uji.id")
	orderNo, token := checkoutOrder(t, client, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1},{"id":%d,"qty":1},{"id":%d,"qty":1}],"email":"multi@uji.id"}`, f1, f2, fEmpty))
	if res := paySimulasi(t, client, ts, orderNo, token); res.StatusCode != http.StatusFound {
		t.Fatalf("paySimulasi = %d", res.StatusCode)
	}

	// halaman pay (publik, ber-token): dua item punya has_download, item kosong tidak
	page := doPage(t, client, fmt.Sprintf("%s/pay/%s?t=%s", ts.URL, orderNo, token))
	items := prop(t, page)["order"].(map[string]any)["items"].([]any)
	downloads := 0
	for _, it := range items {
		if it.(map[string]any)["has_download"] == true {
			downloads++
		}
	}
	if downloads != 2 {
		t.Fatalf("item has_download = %d, mau 2 (produk tanpa berkas tak boleh punya tautan)", downloads)
	}

	// invoice: dua tautan /dl dengan item id berbeda (multi tautan)
	page = doPage(t, client, ts.URL+"/pesanan/"+orderNo)
	items = prop(t, page)["order"].(map[string]any)["items"].([]any)
	urls := map[string]bool{}
	for _, it := range items {
		m := it.(map[string]any)
		if m["has_download"] == true {
			urls[fmt.Sprintf("/dl/%s/%v?t=%s", orderNo, m["id"], token)] = true
		}
	}
	if len(urls) != 2 {
		t.Fatalf("tautan unduh unik di invoice = %d (%v), mau 2", len(urls), urls)
	}

	// kedua tautan benar-benar mengunduh isi masing-masing
	for u := range urls {
		res := doGetRes(t, &http.Client{}, ts.URL+u, nil)
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK || !strings.Contains(string(b), "KONTEN-") {
			t.Errorf("unduh %s = %d %q", u, res.StatusCode, truncate(string(b), 40))
		}
	}
}

// ---- halaman setting admin (SMTP/konten/SEO) + detail order admin + SEO produk ----

func TestE2EAdminSettingsAndDetail(t *testing.T) {
	_, ts, d := testServer(t)
	seedUser(t, d, "boss@uji.id", "adminpass1", "admin")
	admin := loginCookie(t, ts, "boss@uji.id", "adminpass1")
	acsrf := csrfOf(t, admin, ts, "/admin")

	// simpan konten kebijakan + seo
	resC := doPostCSRF(t, admin, ts.URL+"/admin/setting/content",
		`{"privacy":"# Privasi Uji\nIsi privasi dari setting.","terms":"# Ketentuan Uji\nIsi ketentuan."}`, acsrf)
	if loc := resC.Header.Get("Location"); !strings.Contains(loc, "ok=") {
		t.Fatalf("simpan konten = %d | %s", resC.StatusCode, loc)
	}
	doPostCSRF(t, admin, ts.URL+"/admin/setting/seo",
		`{"seo.home.title":"Toko Uji - Home","seo.products.title":"Katalog Toko Uji","seo.product.title":"Beli {title}","seo.product.description":"Deskripsi {title}"}`, acsrf)

	// halaman privacy memakai konten dari setting
	page := doPage(t, newClient(), ts.URL+"/privacy")
	if !strings.Contains(fmt.Sprint(prop(t, page)["content"]), "Isi privasi dari setting.") {
		t.Error("konten privacy setting tidak tampil")
	}
	// landing membawa seo props
	page = doPage(t, newClient(), ts.URL+"/")
	if prop(t, page)["seo"].(map[string]any)["seo.home.title"] != "Toko Uji - Home" {
		t.Error("seo home tidak masuk props")
	}

	// produk dengan meta SEO
	res := multipartPost(t, admin, ts.URL+"/admin/produk", acsrf, map[string]string{
		"slug": "produk-seo", "title": "Produk SEO", "price": "5000", "type": "key",
		"meta_title": "Meta Khusus", "meta_description": "Desc khusus",
	}, "", "", "")
	if res.StatusCode != http.StatusFound {
		t.Fatalf("create produk seo = %d", res.StatusCode)
	}
	page = doPage(t, newClient(), ts.URL+"/produk/produk-seo")
	pd := prop(t, page)["product"].(map[string]any)
	if pd["meta_title"] != "Meta Khusus" || pd["meta_description"] != "Desc khusus" {
		t.Errorf("meta produk = %v %v", pd["meta_title"], pd["meta_description"])
	}

	// order user → admin lihat detail
	uid := seedUser(t, d, "pembeli@uji.id", "rahasia123", "user")
	_ = uid
	pid := seedProduct(t, d, "produk-detail-uji", 20000, "key")
	seedKeys(t, d, pid, 1)
	user := loginCookie(t, ts, "pembeli@uji.id", "rahasia123")
	orderNo, token := checkoutOrder(t, user, ts,
		fmt.Sprintf(`{"items":[{"id":%d,"qty":1}],"email":"pembeli@uji.id"}`, pid))
	if res := paySimulasi(t, user, ts, orderNo, token); res.StatusCode != http.StatusFound {
		t.Fatalf("pay = %d", res.StatusCode)
	}

	// admin lihat detail order user
	page = doPage(t, admin, ts.URL+"/admin/orders/"+orderNo)
	o := prop(t, page)["order"].(map[string]any)
	if o["order_no"] != orderNo || o["email"] != "pembeli@uji.id" {
		t.Errorf("detail order salah: %v %v", o["order_no"], o["email"])
	}
	if len(o["items"].([]any)) != 1 {
		t.Error("item detail order hilang")
	}
	if fmt.Sprint(prop(t, page)["token"]) == "" {
		t.Error("token unduh tidak diberikan ke admin")
	}
	// user lain tak bisa akses halaman admin
	if res := doGetRes(t, user, ts.URL+"/admin/orders/"+orderNo, map[string]string{"X-Inertia": "true"}); res.StatusCode != http.StatusForbidden {
		t.Errorf("detail order utk user biasa = %d, mau 403", res.StatusCode)
	}
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
