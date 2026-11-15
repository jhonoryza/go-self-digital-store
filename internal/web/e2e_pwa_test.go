package web

// PWA aset: manifest dinamis (nama toko + warna tema aktif), service worker,
// offline fallback, favicon & ikon — dilayani dari web/dist hasil build Vite.

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EPwaAssets(t *testing.T) {
	// handler aset PWA membaca web/dist relatif ke akar repo
	t.Chdir(filepath.Join("..", ".."))
	_, ts, d := testServer(t)
	seedUser(t, d, "pwa@uji.id", "adminpass1", "admin")
	admin := loginCookie(t, ts, "pwa@uji.id", "adminpass1")
	acsrf := csrfOf(t, admin, ts, "/admin")

	manifest := func() map[string]any {
		t.Helper()
		res := doGetRes(t, newClient(), ts.URL+"/manifest.webmanifest", nil)
		if res.StatusCode != http.StatusOK || !strings.Contains(res.Header.Get("Content-Type"), "manifest+json") {
			t.Fatalf("manifest = %d %s", res.StatusCode, res.Header.Get("Content-Type"))
		}
		var m map[string]any
		if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
			t.Fatalf("manifest bukan JSON: %v", err)
		}
		return m
	}

	// bawaan: nama toko dari config, warna tema default
	m := manifest()
	if m["name"] != "Toko Uji" || m["start_url"] != "/" || m["display"] != "standalone" {
		t.Errorf("manifest bawaan = %v", m)
	}
	if m["theme_color"] != themeHex[""] {
		t.Errorf("theme_color bawaan = %v, mau %s", m["theme_color"], themeHex[""])
	}

	// manifest dinamis: ganti tema admin → warna ikut berubah
	if res := doPostCSRF(t, admin, ts.URL+"/admin/setting/theme", `{"theme":"neon"}`, acsrf); res.StatusCode != http.StatusFound {
		t.Fatalf("simpan tema = %d", res.StatusCode)
	}
	m = manifest()
	if m["theme_color"] != themeHex["neon"] {
		t.Errorf("theme_color pasca tema = %v, mau %s", m["theme_color"], themeHex["neon"])
	}

	// aset statis dari web/dist (skip bila frontend belum dibangun)
	if _, err := os.Stat("web/dist/sw.js"); err != nil {
		t.Skip("web/dist belum dibangun - jalankan cd web && bun run build")
	}

	// sw.js: javascript, tanpa cache (versi SW harus segar)
	res := doGetRes(t, newClient(), ts.URL+"/sw.js", nil)
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK ||
		!strings.Contains(res.Header.Get("Content-Type"), "text/javascript") ||
		res.Header.Get("Cache-Control") != "no-cache" ||
		!strings.Contains(string(b), "offline.html") {
		t.Errorf("sw.js = %d %s cache=%q", res.StatusCode, res.Header.Get("Content-Type"), res.Header.Get("Cache-Control"))
	}

	// offline.html: halaman fallback
	res = doGetRes(t, newClient(), ts.URL+"/offline.html", nil)
	if res.StatusCode != http.StatusOK || !strings.Contains(res.Header.Get("Content-Type"), "text/html") {
		t.Errorf("offline.html = %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}

	// favicon
	res = doGetRes(t, newClient(), ts.URL+"/favicon.ico", nil)
	if res.StatusCode != http.StatusOK || !strings.Contains(res.Header.Get("Content-Type"), "image/x-icon") {
		t.Errorf("favicon = %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}

	// ikon PWA: png dilayani, non-png & path traversal ditolak
	res = doGetRes(t, newClient(), ts.URL+"/icons/icon-192.png", nil)
	if res.StatusCode != http.StatusOK || !strings.Contains(res.Header.Get("Content-Type"), "image/png") {
		t.Errorf("icon-192 = %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	if res := doGetRes(t, newClient(), ts.URL+"/icons/nope.txt", nil); res.StatusCode != http.StatusNotFound {
		t.Errorf("ikon non-png = %d, mau 404", res.StatusCode)
	}
	if res := doGetRes(t, newClient(), ts.URL+"/icons/%2e%2e%2fgo.mod", nil); res.StatusCode != http.StatusNotFound {
		t.Errorf("traversal ikon = %d, mau 404", res.StatusCode)
	}
}
