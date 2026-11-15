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
	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	neturl "net/url"

	"strings"
	"testing"
	"time"
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
