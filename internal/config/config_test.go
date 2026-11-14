package config

// Test fallback kredensial R2 dari konfigurasi MinIO client — kredensial
// contoh di sini palsu (bukan milik siapa pun).

import (
	"os"
	"path/filepath"
	"testing"
)

func TestR2CredsFromMCJSON(t *testing.T) {
	raw := []byte(`{
	  "aliases": {
	    "gcs": {"url": "https://storage.googleapis.com", "accessKey": "a", "secretKey": "b"},
	    "r2": {"url": "https://account123.r2.cloudflarestorage.com", "accessKey": "AK", "secretKey": "SK"},
	    "play": {"url": "https://play.min.io", "accessKey": "x", "secretKey": "y"}
	  }
	}`)
	ak, sk, ep := r2CredsFromMCJSON(raw)
	if ep != "https://account123.r2.cloudflarestorage.com" || ak != "AK" || sk != "SK" {
		t.Fatalf("json: %q %q %q", ak, sk, ep)
	}
	// tanpa alias r2 → alias pertama ber-endpoint R2
	raw2 := []byte(`{"aliases": {"toko": {"url": "https://xyz.r2.cloudflarestorage.com", "accessKey": "A2", "secretKey": "S2"}}}`)
	ak, sk, ep = r2CredsFromMCJSON(raw2)
	if ep == "" || ak != "A2" || sk != "S2" {
		t.Fatalf("json fallback: %q %q %q", ak, sk, ep)
	}
}

func TestR2CredsFromMCINI(t *testing.T) {
	raw := []byte("[alias r2]\nurl = https://acc.r2.cloudflarestorage.com\naccess_key = INIAK\nsecret_key = INISK\n")
	ak, sk, ep := r2CredsFromMCINI(raw)
	if ep != "https://acc.r2.cloudflarestorage.com" || ak != "INIAK" || sk != "INISK" {
		t.Fatalf("ini: %q %q %q", ak, sk, ep)
	}
}

func TestLoadR2FallbackMC(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".mc")
	os.MkdirAll(dir, 0o700)
	os.WriteFile(filepath.Join(dir, "config.json"),
		[]byte(`{"aliases": {"r2": {"url": "https://mc.r2.cloudflarestorage.com", "accessKey": "MCAK", "secretKey": "MCSK"}}}`), 0o600)

	t.Setenv("HOME", home)
	t.Setenv("STORE_R2_ENDPOINT", "")
	t.Setenv("STORE_R2_ACCESS_KEY", "")
	t.Setenv("STORE_R2_SECRET_KEY", "")
	r := loadR2()
	if !r.Enabled() {
		t.Fatal("R2 tidak aktif dari fallback mc")
	}
	if r.AccessKey != "MCAK" || r.Endpoint != "https://mc.r2.cloudflarestorage.com" {
		t.Fatalf("fallback salah: %+v", r)
	}
	if r.Bucket != "webapp" || r.Prefix != "go-self-digital-store" || r.MaxImages != 5 {
		t.Fatalf("default salah: %+v", r)
	}
}
