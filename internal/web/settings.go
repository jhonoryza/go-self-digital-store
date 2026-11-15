package web

// Setting admin: SMTP runtime (Mailer.Reconfigure), brand (nama + logo di
// R2), konten halaman kebijakan, dan SEO per halaman. Semua tersimpan di
// tabel settings (key-value) agar bisa diubah tanpa deploy.

import (
	"net/http"
	"path"
	"strings"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	"github.com/jhonoryza/go-self-digital-store/internal/mail"
)

// appName nama toko efektif: setting brand.name menimpa config env.
func (s *Server) appName() string {
	if n := getSetting(s.DB, "brand.name"); n != "" {
		return n
	}
	return s.Cfg.AppName
}

// smtpFromSettings susun config SMTP dari settings (fallback ke config env
// untuk field yang kosong) — dipakai saat menyimpan & membangun Mailer.
func (s *Server) smtpFromSettings() mail.Config {
	get := func(key, fallback string) string {
		if v := getSetting(s.DB, key); v != "" {
			return v
		}
		return fallback
	}
	return mail.Config{
		Host: get("smtp.host", s.Cfg.SMTP.Host),
		Port: get("smtp.port", s.Cfg.SMTP.Port),
		User: get("smtp.user", s.Cfg.SMTP.User),
		Pass: get("smtp.pass", s.Cfg.SMTP.Pass),
		From: get("smtp.from", s.Cfg.SMTP.From),
	}
}

// brandLogoURL URL publik logo brand bila diunggah (key di settings).
func (s *Server) brandLogoURL(r *http.Request) string {
	key := getSetting(s.DB, "brand.logo_key")
	if key == "" || s.Images == nil {
		return ""
	}
	return s.imageURL(r, key)
}

// adminSettingPage GET /admin/setting — satu halaman, empat seksi.
func (s *Server) adminSettingPage(w http.ResponseWriter, r *http.Request, u auth.User) {
	smtpCfg := s.smtpFromSettings()
	var contentPrivacy, contentTerms string
	s.DB.QueryRow(`SELECT value FROM settings WHERE key='content.privacy'`).Scan(&contentPrivacy)
	s.DB.QueryRow(`SELECT value FROM settings WHERE key='content.terms'`).Scan(&contentTerms)
	seo := map[string]string{}
	for _, k := range []string{
		"seo.home.title", "seo.home.description",
		"seo.products.title", "seo.products.description",
		"seo.product.title", "seo.product.description",
	} {
		seo[k] = getSetting(s.DB, k)
	}
	var brandName string
	s.DB.QueryRow(`SELECT value FROM settings WHERE key='brand.name'`).Scan(&brandName)
	s.Page(w, r, "Admin/Settings", inertia.Props{
		"smtp": map[string]string{
			"host": smtpCfg.Host, "port": smtpCfg.Port,
			"user": smtpCfg.User, "from": smtpCfg.From,
			// password tidak pernah dikirim balik ke klien
			"has_pass": boolStr(smtpCfg.Pass != ""),
		},
		"brand":          map[string]any{"name": brandName, "logo": s.brandLogoURL(r)},
		"content":        map[string]string{"privacy": contentPrivacy, "terms": contentTerms},
		"seo":            seo,
		"images_enabled": s.Images != nil,
	})
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return ""
}

// adminSettingSMTP POST /admin/setting/smtp — simpan + reconfigure runtime.
// Field kosong berarti "pertahankan nilai lama"; password dikirim hanya bila
// diganti (di UI tampil tersamar).
func (s *Server) adminSettingSMTP(w http.ResponseWriter, r *http.Request, u auth.User) {
	put := func(key, val string) {
		if val != "" {
			setSetting(s.DB, key, val)
		}
	}
	put("smtp.host", strings.TrimSpace(r.PostFormValue("host")))
	put("smtp.port", strings.TrimSpace(r.PostFormValue("port")))
	put("smtp.user", strings.TrimSpace(r.PostFormValue("user")))
	put("smtp.pass", r.PostFormValue("pass"))
	put("smtp.from", strings.TrimSpace(r.PostFormValue("from")))
	cfg := s.smtpFromSettings()
	s.Mail.Reconfigure(cfg)
	audit(s.DB, "setting.smtp", cfg.Host, "oleh "+u.Email)
	s.flashOK(w, r, "/admin/setting", "Konfigurasi SMTP disimpan dan langsung aktif.")
}

// adminSettingSMTPTest POST /admin/setting/smtp/test — kirim email uji ke
// email admin yang sedang login; hasilnya dilaporkan lewat flash.
func (s *Server) adminSettingSMTPTest(w http.ResponseWriter, r *http.Request, u auth.User) {
	cfg := s.smtpFromSettings()
	if cfg.Host == "" {
		s.flashErr(w, r, "/admin/setting", "SMTP belum dikonfigurasi.")
		return
	}
	body := mail.Style(s.appName(), `<p>Ini email uji konfigurasi SMTP. Bila Anda membacanya, konfigurasi berjalan.</p>`)
	if err := s.Mail.TestSend(cfg, u.Email, "Uji SMTP - "+s.appName(), body); err != nil {
		s.flashErr(w, r, "/admin/setting", "Gagal: "+truncate(err.Error(), 120))
		return
	}
	audit(s.DB, "setting.smtp_test", u.Email, cfg.Host)
	s.flashOK(w, r, "/admin/setting", "Email uji terkirim ke "+u.Email+".")
}

// adminSettingBrand POST /admin/setting/brand — nama toko + logo (R2).
func (s *Server) adminSettingBrand(w http.ResponseWriter, r *http.Request, u auth.User) {
	if name := strings.TrimSpace(r.PostFormValue("name")); name != "" {
		setSetting(s.DB, "brand.name", name)
	} else {
		setSetting(s.DB, "brand.name", "")
	}
	removeLogo := r.PostFormValue("remove_logo") == "1"
	file, header, err := r.FormFile("logo")
	switch {
	case err == nil && s.Images == nil:
		s.flashErr(w, r, "/admin/setting", "Penyimpanan gambar (R2) belum aktif - logo tidak dapat diunggah.")
		return
	case err == nil:
		defer file.Close()
		data := make([]byte, 0, header.Size)
		buf := make([]byte, 32*1024)
		for {
			n, err := file.Read(buf)
			data = append(data, buf[:n]...)
			if err != nil {
				break
			}
		}
		sniffed := http.DetectContentType(data)
		exts := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/svg+xml": ".svg"}
		ext, ok := exts[sniffed]
		if !ok {
			s.flashErr(w, r, "/admin/setting", "Logo harus JPG/PNG/WebP/SVG.")
			return
		}
		if len(data) > 1<<20 {
			s.flashErr(w, r, "/admin/setting", "Logo maksimal 1 MB.")
			return
		}
		key := path.Join(s.r2Prefix(), "brand", "logo"+ext)
		if err := s.Images.Put(r.Context(), key, sniffed, data); err != nil {
			logErr("upload logo: %v", err)
			s.flashErr(w, r, "/admin/setting", "Gagal mengunggah logo.")
			return
		}
		if old := getSetting(s.DB, "brand.logo_key"); old != "" && old != key {
			s.Images.Delete(r.Context(), old)
		}
		setSetting(s.DB, "brand.logo_key", key)
		setSetting(s.DB, "brand.name", strings.TrimSpace(r.PostFormValue("name")))
	case removeLogo:
		if old := getSetting(s.DB, "brand.logo_key"); old != "" && s.Images != nil {
			s.Images.Delete(r.Context(), old)
		}
		setSetting(s.DB, "brand.logo_key", "")
	}
	audit(s.DB, "setting.brand", getSetting(s.DB, "brand.name"), "oleh "+u.Email)
	s.flashOK(w, r, "/admin/setting", "Brand disimpan.")
}

// adminSettingContent POST /admin/setting/content — isi privasi & ketentuan.
func (s *Server) adminSettingContent(w http.ResponseWriter, r *http.Request, u auth.User) {
	setSetting(s.DB, "content.privacy", strings.TrimSpace(r.PostFormValue("privacy")))
	setSetting(s.DB, "content.terms", strings.TrimSpace(r.PostFormValue("terms")))
	audit(s.DB, "setting.content", "", "oleh "+u.Email)
	s.flashOK(w, r, "/admin/setting", "Konten halaman kebijakan disimpan.")
}

// adminSettingSEO POST /admin/setting/seo — meta per halaman.
func (s *Server) adminSettingSEO(w http.ResponseWriter, r *http.Request, u auth.User) {
	for _, key := range []string{
		"seo.home.title", "seo.home.description",
		"seo.products.title", "seo.products.description",
		"seo.product.title", "seo.product.description",
	} {
		setSetting(s.DB, key, strings.TrimSpace(r.PostFormValue(key)))
	}
	audit(s.DB, "setting.seo", "", "oleh "+u.Email)
	s.flashOK(w, r, "/admin/setting", "Pengaturan SEO disimpan.")
}

// seoFor meta SEO satu halaman dari settings (kosong = frontend pakai default).
func (s *Server) seoFor(keys ...string) map[string]string {
	out := map[string]string{}
	for _, k := range keys {
		out[k] = getSetting(s.DB, k)
	}
	return out
}
