// Package config memuat konfigurasi dari environment variable ber-prefix STORE_*.
package config

import (
	"os"
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
