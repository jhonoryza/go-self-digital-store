package web

// Blok test dipecah per domain dari e2e_test.go (helper tetap di sana).

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	// status konfigurasi: tripay sudah diisi, mock belum
	if tripay["configured"] != true {
		t.Errorf("tripay configured = %v, mau true", tripay["configured"])
	}
	if mock != nil && mock["configured"] != false {
		t.Errorf("mock configured = %v, mau false", mock["configured"])
	}

	// activate tripay
	res = doPostCSRF(t, admin, ts.URL+"/admin/gateway/activate", `{"provider":"tripay"}`, acsrf)
	if res.StatusCode != http.StatusFound || getSetting(d, "payment_provider") != "tripay" {
		t.Errorf("activate = %d %q", res.StatusCode, getSetting(d, "payment_provider"))
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

// ---- konsol: filter/search/sort/paginasi admin produk & users ----

func TestE2EAdminLists(t *testing.T) {
	_, ts, d := testServer(t)
	seedUser(t, d, "boss@uji.id", "adminpass1", "admin")
	admin := loginCookie(t, ts, "boss@uji.id", "adminpass1")

	// 15 produk: 3 kategori x 2 jenis, harga bertingkat; satu nonaktif
	insert := func(slug, title, category, typ string, price int, active int) {
		_, err := d.Exec(`INSERT INTO products (slug, title, description, price, type, active, category) VALUES (?,?,?,?,?,?,?)`,
			slug, title, "deskripsi "+slug, price, typ, active, category)
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= 15; i++ {
		typ := "file"
		if i%2 == 0 {
			typ = "key"
		}
		active := 1
		if i == 15 {
			active = 0 // satu produk nonaktif utk filter status
		}
		insert(fmt.Sprintf("adm-%02d", i), fmt.Sprintf("Produk Admin %02d", i),
			[]string{"Kategori-A", "Kategori-B", "Kategori-C"}[i%3], typ, i*10000, active)
	}

	// 14 pengguna tambahan: bervariasi role & verifikasi
	for i := 1; i <= 14; i++ {
		role := "user"
		verified := "datetime('now')"
		if i%7 == 0 {
			role = "admin"
		}
		if i%2 == 1 {
			verified = "NULL"
		}
		if _, err := d.Exec(`INSERT INTO users (email, name, password, role, email_verified_at) VALUES (?,?,?,?,`+verified+`)`,
			fmt.Sprintf("user%02d@uji.id", i), fmt.Sprintf("Pengguna %02d", i), "x", role); err != nil {
			t.Fatal(err)
		}
	}

	// ---- admin/produk ----
	// pencarian
	page := doPage(t, admin, ts.URL+"/admin/produk?q=Admin+07")
	if got := prop(t, page)["total"].(float64); got != 1 {
		t.Errorf("q=Admin 07 total = %v, mau 1", got)
	}
	// filter kategori + jenis
	page = doPage(t, admin, ts.URL+"/admin/produk?kategori=Kategori-A&type=key")
	if got := prop(t, page)["total"].(float64); got != 2 { // i=1,4,7,10,13 → A; genap key: 4,10
		t.Errorf("kategori+type total = %v, mau 2", got)
	}
	// filter status nonaktif
	page = doPage(t, admin, ts.URL+"/admin/produk?status=nonaktif")
	if got := prop(t, page)["total"].(float64); got != 1 {
		t.Errorf("status=nonaktif total = %v, mau 1", got)
	}
	// urutan termurah → produk pertama termurah (adm-01, 10rb)
	page = doPage(t, admin, ts.URL+"/admin/produk?sort=termurah")
	first := prop(t, page)["products"].([]any)[0].(map[string]any)
	if first["slug"] != "adm-01" {
		t.Errorf("sort termurah pertama = %v, mau adm-01", first["slug"])
	}
	// paginasi: 15 produk → 2 halaman; page 2 berisi 3
	page = doPage(t, admin, ts.URL+"/admin/produk")
	if prop(t, page)["last_page"].(float64) != 2 {
		t.Errorf("last_page = %v, mau 2", prop(t, page)["last_page"])
	}
	page = doPage(t, admin, ts.URL+"/admin/produk?page=2")
	if got := len(prop(t, page)["products"].([]any)); got != 3 {
		t.Errorf("page 2 = %d produk, mau 3", got)
	}

	// ---- admin/users ----
	// pencarian
	page = doPage(t, admin, ts.URL+"/admin/users?q=user07")
	if got := prop(t, page)["total"].(float64); got != 1 {
		t.Errorf("q=user07 total = %v, mau 1", got)
	}
	// filter role admin (seed boss + user07 + user14)
	page = doPage(t, admin, ts.URL+"/admin/users?role=admin")
	if got := prop(t, page)["total"].(float64); got != 3 {
		t.Errorf("role=admin total = %v, mau 3", got)
	}
	// filter belum verifikasi (ganjil: user01..13 = 7 orang)
	page = doPage(t, admin, ts.URL+"/admin/users?verified=belum")
	if got := prop(t, page)["total"].(float64); got != 7 {
		t.Errorf("verified=belum total = %v, mau 7", got)
	}
	// urutan email A-Z → user01 teratas (boss@ di atasnya? email 'boss@' < 'user01@')
	page = doPage(t, admin, ts.URL+"/admin/users?sort=email&role=user")
	first = prop(t, page)["users"].([]any)[0].(map[string]any)
	if first["email"] != "user01@uji.id" {
		t.Errorf("sort email pertama = %v, mau user01@uji.id", first["email"])
	}
	// paginasi: 15 users (14 + boss) → 2 halaman; page 2 berisi 3
	page = doPage(t, admin, ts.URL+"/admin/users")
	if prop(t, page)["last_page"].(float64) != 2 {
		t.Errorf("users last_page = %v, mau 2", prop(t, page)["last_page"])
	}
	page = doPage(t, admin, ts.URL+"/admin/users?page=2")
	if got := len(prop(t, page)["users"].([]any)); got != 3 {
		t.Errorf("users page 2 = %d, mau 3", got)
	}

	// ---- admin/orders: filter status, search, sort, paginasi ----
	// 5 order: 3 hari ini (2 paid 1 pending), 2 lama (40 & 400 hari)
	fixtures := []struct {
		no, email, status string
		total             int
		hariLalu          int
	}{
		{"ORD-A1", "a1@uji.id", "paid", 100000, 0},
		{"ORD-A2", "a2@uji.id", "pending", 200000, 0},
		{"ORD-A3", "a3@uji.id", "paid", 300000, 0},
		{"ORD-B1", "b1@uji.id", "paid", 400000, 40},
		{"ORD-C1", "c1@uji.id", "paid", 500000, 400},
	}
	for _, f := range fixtures {
		_, err := d.Exec(`INSERT INTO orders (order_no, email, status, subtotal, total, pay_provider, created_at)
			VALUES (?,?,?,?,?,?, datetime('now', ?))`,
			f.no, f.email, f.status, f.total, f.total, "mock", fmt.Sprintf("-%d days", f.hariLalu))
		if err != nil {
			t.Fatal(err)
		}
	}
	page = doPage(t, admin, ts.URL+"/admin/orders?status=paid")
	if got := prop(t, page)["total"].(float64); got != 4 {
		t.Errorf("orders status=paid total = %v, mau 4", got)
	}
	page = doPage(t, admin, ts.URL+"/admin/orders?q=ORD-A2")
	if got := prop(t, page)["total"].(float64); got != 1 {
		t.Errorf("orders q total = %v, mau 1", got)
	}
	page = doPage(t, admin, ts.URL+"/admin/orders?sort=termahal")
	first = prop(t, page)["orders"].([]any)[0].(map[string]any)
	if first["order_no"] != "ORD-C1" {
		t.Errorf("orders termahal pertama = %v, mau ORD-C1", first["order_no"])
	}
	// paginasi 5 order → 1 halaman; pastikan props paginasi ada
	page = doPage(t, admin, ts.URL+"/admin/orders")
	if prop(t, page)["last_page"].(float64) != 1 || prop(t, page)["total"].(float64) != 5 {
		t.Errorf("orders last_page/total = %v/%v, mau 1/5", prop(t, page)["last_page"], prop(t, page)["total"])
	}

	// detail order admin: created_at terisi (regresi "Dibuat:" kosong) dan
	// item tampil setelah order_items diisi
	d.Exec(`INSERT INTO order_items (order_id, product_id, title, price, qty)
		SELECT o.id, p.id, p.title, 100000, 1 FROM orders o, products p
		WHERE o.order_no='ORD-A1' AND p.slug='adm-01'`)
	page = doPage(t, admin, ts.URL+"/admin/orders/ORD-A1")
	od := prop(t, page)["order"].(map[string]any)
	if od["created_at"] == "" {
		t.Error("detail order admin: created_at kosong")
	}
	if items, ok := od["items"].([]any); !ok || len(items) != 1 {
		t.Errorf("detail order admin items = %v, mau 1", od["items"])
	}

	// ---- admin/tiket: filter status + search + sort ----
	var bossID int64
	d.QueryRow(`SELECT id FROM users WHERE email='boss@uji.id'`).Scan(&bossID)
	tiketFixtures := []struct {
		subj, status string
	}{
		{"Kunci tidak masuk", "open"},
		{"Tanya invoice", "answered"},
		{"Refund please", "closed"},
	}
	for _, tf := range tiketFixtures {
		if _, err := d.Exec(`INSERT INTO tickets (user_id, order_no, subject, status) VALUES (?,?,?,?)`,
			bossID, "ORD-A1", tf.subj, tf.status); err != nil {
			t.Fatal(err)
		}
	}
	page = doPage(t, admin, ts.URL+"/admin/tiket?status=open")
	if got := prop(t, page)["total"].(float64); got != 1 {
		t.Errorf("tiket status=open total = %v, mau 1", got)
	}
	page = doPage(t, admin, ts.URL+"/admin/tiket?q=invoice")
	if got := prop(t, page)["total"].(float64); got != 1 {
		t.Errorf("tiket q=invoice total = %v, mau 1", got)
	}
	page = doPage(t, admin, ts.URL+"/admin/tiket?sort=subjek")
	first = prop(t, page)["tickets"].([]any)[0].(map[string]any)
	if first["subject"] != "Kunci tidak masuk" {
		t.Errorf("tiket subjek pertama = %v, mau Kunci tidak masuk", first["subject"])
	}

	// ---- dashboard admin: periode memfilter kartu + grafik ----
	page = doPage(t, admin, ts.URL+"/admin")
	if got := prop(t, page)["periode"]; got != "" {
		t.Errorf("periode default = %v, mau kosong", got)
	}
	if got := prop(t, page)["series"].([]any); len(got) != 12 {
		t.Errorf("series semua = %d bucket, mau 12", len(got))
	}
	// order terbaru (recent) sudah dihapus dari props
	if _, ok := prop(t, page)["recent"]; ok {
		t.Error("props recent masih ada, seharusnya dihapus")
	}
	// periode mingguan: 7 bucket, hanya order 3 hari ini terhitung
	page = doPage(t, admin, ts.URL+"/admin?periode=mingguan")
	if got := len(prop(t, page)["series"].([]any)); got != 7 {
		t.Errorf("series mingguan = %d bucket, mau 7", got)
	}
	st := prop(t, page)["stats"].(map[string]any)
	if st["orders_paid"].(float64) != 2 {
		t.Errorf("mingguan orders_paid = %v, mau 2", st["orders_paid"])
	}
	// periode bulanan (30 hari): ORD-B1 (40 hari) tidak terhitung
	page = doPage(t, admin, ts.URL+"/admin?periode=bulanan")
	st = prop(t, page)["stats"].(map[string]any)
	if st["orders_paid"].(float64) != 2 {
		t.Errorf("bulanan orders_paid = %v, mau 2", st["orders_paid"])
	}
	if got := len(prop(t, page)["series"].([]any)); got != 30 {
		t.Errorf("series bulanan = %d bucket, mau 30", got)
	}
	// semua: revenue total 1.3jt (A1..C1 lunas)
	page = doPage(t, admin, ts.URL+"/admin")
	st = prop(t, page)["stats"].(map[string]any)
	if st["revenue"].(float64) != 1300000 {
		t.Errorf("revenue semua = %v, mau 1300000", st["revenue"])
	}
	// pie memuat status paid & pending
	pie := prop(t, page)["pie"].([]any)
	statusPie := map[string]float64{}
	for _, it := range pie {
		m := it.(map[string]any)
		statusPie[m["status"].(string)] = m["count"].(float64)
	}
	if statusPie["paid"] != 4 || statusPie["pending"] != 1 {
		t.Errorf("pie = %v, mau paid 4 pending 1", statusPie)
	}
}

// ---- setting tema toko: simpan, shell membawa data-theme, validasi ----

func TestE2EStoreTheme(t *testing.T) {
	_, ts, d := testServer(t)
	seedUser(t, d, "boss@uji.id", "adminpass1", "admin")
	admin := loginCookie(t, ts, "boss@uji.id", "adminpass1")
	acsrf := csrfOf(t, admin, ts, "/admin")

	// tema tidak valid ditolak
	res := doPostCSRF(t, admin, ts.URL+"/admin/setting/theme", `{"theme":"hacker-purple"}`, acsrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("tema ngawur = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// simpan tema brutalism → tersimpan, shell + props halaman membawanya
	if res := doPostCSRF(t, admin, ts.URL+"/admin/setting/theme", `{"theme":"brutalism"}`, acsrf); res.StatusCode != http.StatusFound {
		t.Fatalf("simpan tema = %d", res.StatusCode)
	}
	if got := getSetting(d, "brand.theme"); got != "brutalism" {
		t.Errorf("brand.theme = %q, mau brutalism", got)
	}
	page := doPage(t, admin, ts.URL+"/admin/setting")
	if got := prop(t, page)["theme"]; got != "brutalism" {
		t.Errorf("props theme = %v, mau brutalism", got)
	}
	// shell HTML memuat atribut data-theme (diterapkan sebelum paint)
	shell := doGetRes(t, newClient(), ts.URL+"/", nil)
	b, _ := io.ReadAll(shell.Body)
	if !strings.Contains(string(b), `data-theme="brutalism"`) {
		t.Error("shell tanpa data-theme brutalism")
	}

	// kembali ke bawaan → atribut kosong
	if res := doPostCSRF(t, admin, ts.URL+"/admin/setting/theme", `{"theme":""}`, acsrf); res.StatusCode != http.StatusFound {
		t.Fatalf("reset tema = %d", res.StatusCode)
	}
	shell = doGetRes(t, newClient(), ts.URL+"/", nil)
	b, _ = io.ReadAll(shell.Body)
	if strings.Contains(string(b), `data-theme="brutalism"`) {
		t.Error("shell masih memuat tema lama")
	}
}

// ---- SMTP runtime + brand (nama & logo) ----

func TestE2EAdminSmtpAndBrand(t *testing.T) {
	s, ts, d := testServer(t)
	s.Images = newStubStore() // logo diunggah ke penyimpanan stub
	seedUser(t, d, "setting@uji.id", "adminpass1", "admin")
	admin := loginCookie(t, ts, "setting@uji.id", "adminpass1")
	acsrf := csrfOf(t, admin, ts, "/admin")

	// test-send tanpa konfigurasi → ditolak dengan flash err
	res := doPostCSRF(t, admin, ts.URL+"/admin/setting/smtp/test", `{}`, acsrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
		t.Fatalf("uji SMTP tanpa config = %d %s", res.StatusCode, res.Header.Get("Location"))
	}

	// simpan SMTP runtime → settings tersimpan (Reconfigure dipanggil handler)
	res = doPostCSRF(t, admin, ts.URL+"/admin/setting/smtp",
		`{"host":"127.0.0.1","port":"1","user":"uji","pass":"rahasia","from":"dev@toko.local"}`, acsrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "ok=") {
		t.Fatalf("simpan SMTP = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	if got := getSetting(d, "smtp.host"); got != "127.0.0.1" || getSetting(d, "smtp.port") != "1" {
		t.Errorf("smtp tersimpan = %q:%q", getSetting(d, "smtp.host"), getSetting(d, "smtp.port"))
	}

	// test-send ke host mati (port 1) → gagal + tercatat error di mail_log
	res = doPostCSRF(t, admin, ts.URL+"/admin/setting/smtp/test", `{}`, acsrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "err=") {
		t.Errorf("uji SMTP mati = %d %s, mau flash err", res.StatusCode, res.Header.Get("Location"))
	}
	var nerr int
	d.QueryRow(`SELECT COUNT(*) FROM mail_log WHERE status='error' AND to_email='setting@uji.id'`).Scan(&nerr)
	if nerr < 1 {
		t.Errorf("mail_log error = %d, mau >= 1", nerr)
	}

	// brand: nama saja (tanpa berkas)
	res = doPostCSRF(t, admin, ts.URL+"/admin/setting/brand", `{"name":"Toko Baru Uji"}`, acsrf)
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "ok=") {
		t.Fatalf("brand nama = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	if got := getSetting(d, "brand.name"); got != "Toko Baru Uji" {
		t.Errorf("brand.name = %q", got)
	}

	// brand: unggah logo PNG → key tersimpan + objek masuk stub store
	res = multipartPost(t, admin, ts.URL+"/admin/setting/brand", acsrf,
		map[string]string{"name": "Toko Logo Uji"}, "logo", "logo.png", string(pngBytes))
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "ok=") {
		t.Fatalf("brand logo = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	key := getSetting(d, "brand.logo_key")
	if !strings.Contains(key, "brand/logo.png") {
		t.Fatalf("brand.logo_key = %q, mau berakhiran brand/logo.png", key)
	}
	if len(s.Images.(*stubStore).objects[key]) == 0 {
		t.Error("logo tidak masuk penyimpanan")
	}

	// hapus logo → key dikosongkan + objek dibuang
	res = multipartPost(t, admin, ts.URL+"/admin/setting/brand", acsrf,
		map[string]string{"name": "Toko Logo Uji", "remove_logo": "1"}, "", "", "")
	if res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "ok=") {
		t.Fatalf("brand hapus logo = %d %s", res.StatusCode, res.Header.Get("Location"))
	}
	if got := getSetting(d, "brand.logo_key"); got != "" {
		t.Errorf("brand.logo_key pasca hapus = %q, mau kosong", got)
	}
	if _, ok := s.Images.(*stubStore).objects[key]; ok {
		t.Error("objek logo lama tidak dihapus dari penyimpanan")
	}
}
