package web

// Blok test dipecah per domain dari e2e_test.go (helper tetap di sana).

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

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

	// detail produk aktif; produk tanpa gambar tetap membawa images []
	page = doPage(t, newClient(), ts.URL+"/produk/unggulan-uji")
	pd := prop(t, page)["product"].(map[string]any)
	if pd["price"].(float64) != 99000 || pd["type"] != "file" {
		t.Errorf("detail = %v", pd)
	}
	if imgs, ok := prop(t, page)["images"].([]any); !ok || len(imgs) != 0 {
		t.Errorf("images = %v, mau array kosong (bukan null)", prop(t, page)["images"])
	}

	// detail nonaktif / tidak ada → 404 dengan halaman kustom
	res = doGetRes(t, newClient(), ts.URL+"/produk/mati-uji", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("produk nonaktif = %d, mau 404", res.StatusCode)
	}
	res = doGetRes(t, newClient(), ts.URL+"/produk/tak-ada", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("produk tak ada = %d, mau 404", res.StatusCode)
	}
	b404, _ := io.ReadAll(res.Body)
	// shell Inertia membawa komponen 404 (teks dirender React di klien)
	if !strings.Contains(string(b404), "Public/NotFound") {
		t.Error("404 tanpa komponen kustom (Public/NotFound)")
	}
	// XHR Inertia → JSON 404
	res = doGetRes(t, newClient(), ts.URL+"/produk/tak-ada", map[string]string{"X-Inertia": "true"})
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("404 XHR = %d, mau 404", res.StatusCode)
	}

	// robots.txt melarang area privat & mengumumkan sitemap
	res = doGetRes(t, newClient(), ts.URL+"/robots.txt", nil)
	rb, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(rb), "Disallow: /admin") || !strings.Contains(string(rb), "Sitemap:") {
		t.Errorf("robots.txt = %s", rb)
	}
	// sitemap.xml memuat produk aktif, bukan yang nonaktif
	res = doGetRes(t, newClient(), ts.URL+"/sitemap.xml", nil)
	sb2, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(sb2), "/produk/unggulan-uji") || strings.Contains(string(sb2), "/produk/mati-uji") {
		t.Errorf("sitemap.xml = %s", sb2)
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

// ---- katalog: pencarian, filter kategori/tag/harga, urutan ----

func TestE2ECatalogFilters(t *testing.T) {
	_, ts, d := testServer(t)
	insert := func(slug, title, category, tagsJSON string, price int) {
		_, err := d.Exec(`INSERT INTO products (slug, title, description, price, type, category, tags) VALUES (?,?,?,?,?,?,?)`,
			slug, title, "deskripsi "+slug, price, "file", category, tagsJSON)
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("kunci-editor-foto", "Kunci Editor Foto", "Preset", `["foto","premium"]`, 100000)
	insert("kunci-editor-video", "Kunci Editor Video", "Preset", `["video","premium"]`, 200000)
	insert("buku-panduan-go", "Buku Panduan Go", "E-Book", `["belajar"]`, 50000)
	insert("template-undangan", "Template Undangan", "Template", `["acara","foto"]`, 75000)
	insert("buku-panduan-sql", "Buku Panduan SQL", "E-Book", `["belajar","database"]`, 150000)
	insert("template-undangan-nikah", "Template Undangan Nikah", "Template", `["acara"]`, 300000)

	slugsOf := func(t *testing.T, path string) map[string]bool {
		t.Helper()
		page := doPage(t, newClient(), ts.URL+path)
		raw, ok := prop(t, page)["products"].([]any)
		if !ok {
			t.Fatalf("%s tanpa props products", path)
		}
		out := map[string]bool{}
		for _, it := range raw {
			out[it.(map[string]any)["slug"].(string)] = true
		}
		return out
	}

	// opsi filter: kategori & tag unik terkumpul
	page := doPage(t, newClient(), ts.URL+"/produk")
	p := prop(t, page)
	if got := p["total"].(float64); got != 6 {
		t.Errorf("total = %v, mau 6", got)
	}
	if got := len(p["categories"].([]any)); got != 3 {
		t.Errorf("categories = %d item, mau 3", got)
	}
	if got := len(p["tags"].([]any)); got != 6 {
		t.Errorf("tags = %d item, mau 6", got)
	}

	// pencarian nama (parsial, case-insensitive)
	if got := slugsOf(t, "/produk?q=Buku"); len(got) != 2 || !got["buku-panduan-go"] || !got["buku-panduan-sql"] {
		t.Errorf("q=Buku -> %v", got)
	}
	// wildcard LIKE di-escape, bukan diperlakukan sebagai pola (0 hasil)
	page = doPage(t, newClient(), ts.URL+"/produk?q=%25")
	if got := prop(t, page)["total"].(float64); got != 0 {
		t.Errorf("q=%% total = %v, mau 0", got)
	}
	// filter kategori
	if got := slugsOf(t, "/produk?kategori=E-Book"); len(got) != 2 || !got["buku-panduan-go"] {
		t.Errorf("kategori -> %v", got)
	}
	// filter tag
	got := slugsOf(t, "/produk?tag=foto")
	if len(got) != 2 || !got["kunci-editor-foto"] || !got["template-undangan"] {
		t.Errorf("tag=foto -> %v", got)
	}
	// rentang harga (inklusif)
	if got := slugsOf(t, "/produk?harga_min=100000&harga_max=200000"); len(got) != 3 {
		t.Errorf("harga 100rb-200rb -> %v", got)
	}

	// urutan: produk pertama tiap sort
	firstOf := func(t *testing.T, path string) string {
		t.Helper()
		page := doPage(t, newClient(), ts.URL+path)
		raw := prop(t, page)["products"].([]any)
		if len(raw) == 0 {
			t.Fatalf("%s tanpa produk", path)
		}
		return raw[0].(map[string]any)["slug"].(string)
	}
	cases := map[string]string{
		"/produk?sort=termurah":                 "buku-panduan-go",
		"/produk?sort=termahal":                 "template-undangan-nikah",
		"/produk?sort=terbaru":                  "template-undangan-nikah", // id terakhir
		"/produk?sort=terlama":                  "kunci-editor-foto",       // id pertama
		"/produk?kategori=E-Book&sort=termahal": "buku-panduan-sql",        // filter + sort
	}
	for path, want := range cases {
		if got := firstOf(t, path); got != want {
			t.Errorf("%s pertama = %s, mau %s", path, got, want)
		}
	}

	// filter + paginasi: 12 produk Preset tambahan -> 14 total, 2 halaman
	for i := 0; i < 12; i++ {
		insert(fmt.Sprintf("preset-extra-%02d", i), fmt.Sprintf("Preset Extra %02d", i), "Preset", `["extra"]`, 10000+i)
	}
	page = doPage(t, newClient(), ts.URL+"/produk?kategori=Preset")
	p = prop(t, page)
	if p["total"].(float64) != 14 || p["last_page"].(float64) != 2 {
		t.Errorf("filter+paginasi: total=%v last_page=%v", p["total"], p["last_page"])
	}
	if got := len(p["products"].([]any)); got != 12 {
		t.Errorf("halaman 1 filter = %d, mau 12", got)
	}
	if got := len(slugsOf(t, "/produk?kategori=Preset&page=2")); got != 2 {
		t.Errorf("halaman 2 filter = %d, mau 2", got)
	}

	// admin membuat produk dengan kategori + tags (input koma, duplikat dilebur)
	seedUser(t, d, "boss@uji.id", "adminpass1", "admin")
	admin := loginCookie(t, ts, "boss@uji.id", "adminpass1")
	acsrf := csrfOf(t, admin, ts, "/admin")
	res := multipartPost(t, admin, ts.URL+"/admin/produk", acsrf, map[string]string{
		"slug": "produk-tag-admin", "title": "Produk Tag Admin", "description": "uji taksonomi",
		"price": "42000", "type": "link", "ext_url": "https://contoh.id/x",
		"kategori": "Uji Kategori", "tags": "TagA, taga, TagB , ",
	}, "", "", "")
	if res.StatusCode < 300 || res.StatusCode > 399 {
		t.Errorf("create produk admin = %d, mau 3xx", res.StatusCode)
	}
	var catDB string
	d.QueryRow(`SELECT category FROM products WHERE slug='produk-tag-admin'`).Scan(&catDB)
	if catDB != "Uji Kategori" {
		t.Errorf("kategori tersimpan = %q", catDB)
	}
	// tag duplikat case-insensitive dilebur jadi 2
	var nTags int
	d.QueryRow(`SELECT json_array_length(tags) FROM products WHERE slug='produk-tag-admin'`).Scan(&nTags)
	if nTags != 2 {
		t.Errorf("jumlah tag = %d, mau 2 (duplikat dilebur)", nTags)
	}
	// produk baru ditemukan via filter kategori & tag
	if got := slugsOf(t, "/produk?kategori=Uji%20Kategori"); !got["produk-tag-admin"] {
		t.Errorf("filter kategori produk baru -> %v", got)
	}
	if got := slugsOf(t, "/produk?tag=TagA"); !got["produk-tag-admin"] {
		t.Errorf("filter tag produk baru -> %v", got)
	}
}
