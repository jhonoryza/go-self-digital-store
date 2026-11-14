package web

// Test fitur gambar produk: unggah/daftar/hapus lewat stub imageStore (tanpa
// R2 nyata), batas 5 gambar, validasi tipe, proxy, dan galeri di props.

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubStore imageStore di memori untuk test.
type stubStore struct{ objects map[string][]byte }

func newStubStore() *stubStore { return &stubStore{objects: map[string][]byte{}} }

func (s *stubStore) Put(_ context.Context, key, _ string, data []byte) error {
	s.objects[key] = data
	return nil
}
func (s *stubStore) Delete(_ context.Context, key string) error { delete(s.objects, key); return nil }
func (s *stubStore) Get(_ context.Context, key string) (string, []byte, error) {
	data, ok := s.objects[key]
	if !ok {
		return "", nil, fmt.Errorf("objek tak ada: %s", key)
	}
	return "image/png", data, nil
}

// pngBytes PNG 1x1 minimal untuk sniffing.
var pngBytes = []byte{
	0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T', 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
}

// multipartImages susun body multipart dengan n file "images".
func multipartImages(t *testing.T, n int, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for i := 0; i < n; i++ {
		fw, _ := w.CreateFormFile("images", fmt.Sprintf("g%d.png", i+1))
		fw.Write(content)
	}
	w.Close()
	return &buf, w.FormDataContentType()
}

func TestE2EProductImages(t *testing.T) {
	s, ts, d := testServer(t)
	s.Images = newStubStore() // injeksi stub (config test tanpa R2)
	uid := seedUser(t, d, "admin-img@uji.id", "adminpass1", "admin")
	_ = uid
	admin := loginCookie(t, ts, "admin-img@uji.id", "adminpass1")
	acsrf := csrfOf(t, admin, ts, "/admin")

	// produk key untuk pengujian
	pid := seedProduct(t, d, "produk-gambar", 50000, "key")

	postImages := func(n int) *http.Response {
		body, ctype := multipartImages(t, n, pngBytes)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/admin/produk/%d/images", ts.URL, pid), body)
		req.Header.Set("Content-Type", ctype)
		req.Header.Set("X-Inertia", "true")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("Origin", ts.URL)
		req.Header.Set("X-CSRF-Token", acsrf)
		res, err := admin.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { res.Body.Close() })
		return res
	}

	// tanpa R2 (server segar tanpa stub) → flash error jelas
	sNoStore, tsNo, _ := testServer(t)
	sNoStore.Images = nil
	if res := doGetRes(t, newClient(), tsNo.URL+"/images/apa/aja.png", nil); res.StatusCode != http.StatusNotFound {
		t.Errorf("proxy tanpa store = %d, mau 404", res.StatusCode)
	}

	// unggah 5 gambar → sukses (flash ok, bukan err)
	if res := postImages(5); res.StatusCode != http.StatusFound || !strings.Contains(res.Header.Get("Location"), "ok=") {
		b, _ := io0(res)
		t.Fatalf("unggah 5 = %d: %s | Location: %s", res.StatusCode, b, res.Header.Get("Location"))
	}
	var count int
	d.QueryRow(`SELECT COUNT(*) FROM product_images WHERE product_id=?`, pid).Scan(&count)
	if count != 5 {
		t.Fatalf("gambar = %d, mau 5", count)
	}

	// ke-6 ditolak
	if res := postImages(1); res.StatusCode != http.StatusFound {
		t.Fatalf("unggah ke-6 = %d, mau 302 (flash err)", res.StatusCode)
	}
	d.QueryRow(`SELECT COUNT(*) FROM product_images WHERE product_id=?`, pid).Scan(&count)
	if count != 5 {
		t.Errorf("gambar setelah ke-6 = %d, mau tetap 5", count)
	}

	// props detail publik memuat 5 URL; kartu katalog memuat image_url pertama
	page := doPage(t, newClient(), ts.URL+"/produk/produk-gambar")
	images := prop(t, page)["images"].([]any)
	if len(images) != 5 {
		t.Fatalf("props images = %d, mau 5", len(images))
	}
	firstURL := fmt.Sprint(images[0].(map[string]any)["url"])
	if !strings.Contains(firstURL, "/images/") {
		t.Errorf("url gambar = %q, mau proxy /images/", firstURL)
	}
	list := doPage(t, newClient(), ts.URL+"/produk")
	products := prop(t, list)["products"].([]any)
	var cardURL string
	for _, p := range products {
		if m := p.(map[string]any); m["slug"] == "produk-gambar" {
			cardURL = fmt.Sprint(m["image_url"])
		}
	}
	if cardURL == "" || !strings.Contains(cardURL, "/images/") {
		t.Errorf("image_url kartu = %q", cardURL)
	}

	// proxy menyajikan isi objek dengan cache immutable
	res := doGetRes(t, newClient(), firstURL, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("proxy = %d, mau 200", res.StatusCode)
	}
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("cache-control = %q", cc)
	}

	// key di luar prefix app ditolak
	res = doGetRes(t, newClient(), ts.URL+"/images/app-lain/rahasia.png", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("proxy lintas prefix = %d, mau 404", res.StatusCode)
	}

	// tipe tak didukung ditolak
	body, ctype := multipartImages(t, 1, []byte("bukan gambar"))
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/admin/produk/%d/images", ts.URL, pid), body)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("X-CSRF-Token", acsrf)
	res2, err := admin.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res2.Body.Close()
	d.QueryRow(`SELECT COUNT(*) FROM product_images WHERE product_id=?`, pid).Scan(&count)
	if count != 5 {
		t.Errorf("gambar setelah file ngawur = %d, mau tetap 5", count)
	}

	// hapus satu → 4
	var imgID int64
	d.QueryRow(`SELECT id FROM product_images WHERE product_id=? ORDER BY id LIMIT 1`, pid).Scan(&imgID)
	if res := doPostCSRF(t, admin, fmt.Sprintf("%s/admin/produk/%d/images/%d/delete", ts.URL, pid, imgID), `{}`, acsrf); res.StatusCode != http.StatusFound {
		t.Fatalf("hapus gambar = %d, mau 302", res.StatusCode)
	}
	d.QueryRow(`SELECT COUNT(*) FROM product_images WHERE product_id=?`, pid).Scan(&count)
	if count != 4 {
		t.Errorf("gambar setelah hapus = %d, mau 4", count)
	}

	// unggah lagi 1 → 5 (slot terpakai kembali)
	if res := postImages(1); res.StatusCode != http.StatusFound {
		t.Fatalf("unggah ulang = %d", res.StatusCode)
	}
	d.QueryRow(`SELECT COUNT(*) FROM product_images WHERE product_id=?`, pid).Scan(&count)
	if count != 5 {
		t.Errorf("gambar setelah isi ulang = %d, mau 5", count)
	}
}

// io0 baca body sebagai string helper kecil.
func io0(res *http.Response) (string, error) {
	buf := new(bytes.Buffer)
	if res.Body != nil {
		_, err := buf.ReadFrom(res.Body)
		res.Body.Close()
		if err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

// pastikan httptest terpakai (import guard)
var _ = httptest.NewServer
