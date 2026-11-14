// Package config memuat konfigurasi dari environment variable ber-prefix STORE_*.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SMTP konfigurasi relay email (Brevo: smtp-relay.brevo.com:587 STARTTLS,
// user = login SMTP, pass = SMTP key — bukan API key).
type SMTP struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

// Enabled true bila host SMTP diisi — app tetap jalan tanpa email.
func (s SMTP) Enabled() bool { return s.Host != "" }

// R2 konfigurasi bucket Cloudflare R2 untuk gambar produk. Kredensial bisa
// dari env atau (fallback dev) dari ~/.mc/config.json alias r2 — kredensial
// tidak pernah ditulis ke repo.
type R2 struct {
	Endpoint   string // https://<account>.r2.cloudflarestorage.com
	AccessKey  string
	SecretKey  string
	Bucket     string // default "webapp"
	Prefix     string // prefix path per-app di bucket bersama, default "go-self-digital-store"
	PublicBase string // opsional domain publik bucket (r2.dev / custom); kosong = proxy /images/ lewat API
	MaxImages  int    // maks gambar per produk, default 5
}

// Enabled true bila kredensial lengkap.
func (r R2) Enabled() bool { return r.Endpoint != "" && r.AccessKey != "" && r.SecretKey != "" }

// Config seluruh konfigurasi runtime.
type Config struct {
	Addr           string   // listen address
	Dir            string   // data dir: db, files/, uploads/, backups/
	BaseURL        string   // URL publik API (tautan di email, unduhan); kosong = pakai request host
	AppName        string   // nama toko (page title, email)
	CorsOrigins    []string // origin frontend (CF Pages) yang boleh memanggil API; kosong = same-origin
	CookieSecure   bool     // force Secure cookie (otomatis true bila BaseURL https)
	CookieSameSite string   // "lax" (default same-origin) | "none" (wajib utk split-origin CF Pages)
	TrustedProxy   bool     // percaya X-Forwarded-For (reverse proxy sendiri)
	PaymentSecret  string   // HMAC webhook adapter kustom (opsional)
	DevLogin       bool     // aktifkan /dev/login (masuk sebagai user/admin tanpa password) — JANGAN di produksi
	R2             R2
	SMTP           SMTP
}

// Load baca env, isi default yang aman untuk dev lokal.
func Load() Config {
	secure := envBool("STORE_COOKIE_SECURE", false) || strings.HasPrefix(env("STORE_BASE_URL", ""), "https://")
	sameSite := strings.ToLower(env("STORE_COOKIE_SAMESITE", ""))
	if sameSite != "none" && sameSite != "lax" {
		// split-origin (CORS origins diisi) menuntut SameSite=None agar cookie
		// terkirim pada XHR lintas origin; same-origin cukup Lax.
		if len(corsOrigins()) > 0 {
			sameSite = "none"
		} else {
			sameSite = "lax"
		}
		if secure && sameSite == "none" {
			secure = true
		}
	}
	return Config{
		Addr:           env("STORE_ADDR", ":8080"),
		Dir:            env("STORE_DIR", "."),
		BaseURL:        strings.TrimRight(env("STORE_BASE_URL", ""), "/"),
		AppName:        env("STORE_NAME", "Toko Digital"),
		CorsOrigins:    corsOrigins(),
		CookieSecure:   secure,
		CookieSameSite: sameSite,
		TrustedProxy:   envBool("STORE_TRUSTED_PROXY", false),
		PaymentSecret:  env("STORE_PAYMENT_SECRET", ""),
		DevLogin:       envBool("STORE_DEV_LOGIN", false),
		R2:             loadR2(),
		SMTP: SMTP{
			Host: env("STORE_SMTP_HOST", ""),
			Port: env("STORE_SMTP_PORT", "587"),
			User: env("STORE_SMTP_USER", ""),
			Pass: env("STORE_SMTP_PASS", ""),
			From: env("STORE_SMTP_FROM", ""),
		},
	}
}

func corsOrigins() []string {
	raw := strings.TrimSpace(env("STORE_CORS_ORIGINS", ""))
	if raw == "" {
		return nil
	}
	var out []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(strings.TrimRight(o, "/")); o != "" {
			out = append(out, o)
		}
	}
	return out
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

// loadR2 susun konfigurasi R2: env STORE_R2_* menang, lalu fallback parsing
// ~/.mc/config.json (atau ~/.mc/config) — ambil alias "r2" atau alias pertama
// yang endpoint-nya Cloudflare R2. Kosong berarti fitur gambar mati.
func loadR2() R2 {
	r := R2{
		Endpoint:   env("STORE_R2_ENDPOINT", ""),
		AccessKey:  env("STORE_R2_ACCESS_KEY", ""),
		SecretKey:  env("STORE_R2_SECRET_KEY", ""),
		Bucket:     env("STORE_R2_BUCKET", "webapp"),
		Prefix:     env("STORE_R2_PREFIX", "go-self-digital-store"),
		PublicBase: strings.TrimRight(env("STORE_R2_PUBLIC_BASE", ""), "/"),
		MaxImages:  5,
	}
	if r.Endpoint == "" || r.AccessKey == "" || r.SecretKey == "" {
		if ak, sk, ep := r2CredsFromMC(); ep != "" {
			if r.Endpoint == "" {
				r.Endpoint = ep
			}
			if r.AccessKey == "" {
				r.AccessKey = ak
			}
			if r.SecretKey == "" {
				r.SecretKey = sk
			}
		}
	}
	if r.MaxImages < 1 {
		r.MaxImages = 5
	}
	return r
}

// r2CredsFromMC baca konfigurasi MinIO client (~/.mc/config.json format JSON
// atau ~/.mc/config format INI) tanpa pernah mencatat secret ke log.
func r2CredsFromMC() (accessKey, secretKey, endpoint string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", ""
	}
	candidates := []string{
		filepath.Join(home, ".mc", "config.json"),
		filepath.Join(home, ".mc", "config"),
	}
	for _, path := range candidates {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if ak, sk, ep := r2CredsFromMCJSON(raw); ep != "" {
			return ak, sk, ep
		}
		if ak, sk, ep := r2CredsFromMCINI(raw); ep != "" {
			return ak, sk, ep
		}
	}
	return "", "", ""
}

// r2CredsFromMCJSON parse format JSON mc (`{"aliases": {"r2": {...}}}`).
func r2CredsFromMCJSON(raw []byte) (string, string, string) {
	var doc struct {
		Aliases map[string]struct {
			URL       string `json:"url"`
			AccessKey string `json:"accessKey"`
			SecretKey string `json:"secretKey"`
		} `json:"aliases"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return "", "", ""
	}
	if a, ok := doc.Aliases["r2"]; ok && isR2Endpoint(a.URL) {
		return a.AccessKey, a.SecretKey, strings.TrimRight(a.URL, "/")
	}
	for _, a := range doc.Aliases {
		if isR2Endpoint(a.URL) {
			return a.AccessKey, a.SecretKey, strings.TrimRight(a.URL, "/")
		}
	}
	return "", "", ""
}

// r2CredsFromMCINI parse format INI mc lama (`[alias r2]` + kunci
// url/access_key/secret_key) — parser manual minimal, tanpa dependensi.
func r2CredsFromMCINI(raw []byte) (string, string, string) {
	type alias struct{ url, ak, sk string }
	aliases := map[string]alias{}
	section := ""
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || !strings.HasPrefix(section, "alias") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(section, "alias"))
		a := aliases[name]
		switch strings.TrimSpace(key) {
		case "url":
			a.url = strings.TrimSpace(val)
		case "access_key":
			a.ak = strings.TrimSpace(val)
		case "secret_key":
			a.sk = strings.TrimSpace(val)
		}
		aliases[name] = a
	}
	if a, ok := aliases["r2"]; ok && isR2Endpoint(a.url) {
		return a.ak, a.sk, strings.TrimRight(a.url, "/")
	}
	for _, a := range aliases {
		if isR2Endpoint(a.url) {
			return a.ak, a.sk, strings.TrimRight(a.url, "/")
		}
	}
	return "", "", ""
}

// isR2Endpoint true bila URL menunjuk Cloudflare R2 S3 API.
func isR2Endpoint(u string) bool {
	return strings.Contains(u, "r2.cloudflarestorage.com")
}
