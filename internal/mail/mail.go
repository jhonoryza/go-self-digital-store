// Package mail kirim email via SMTP relay (Brevo) dengan STARTTLS oportunistik,
// setiap upaya tercatat ke tabel mail_log.
package mail

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"
)

// Config kredensial SMTP (dari config.Config.SMTP).
type Config struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

// Mailer kirim email + catat hasil ke mail_log. Config bisa diganti saat
// runtime (halaman setting admin) — dibaca di balik mutex.
type Mailer struct {
	db  *sql.DB
	mu  sync.RWMutex
	cfg Config
}

// Reconfigure ganti konfigurasi SMTP tanpa restart (dipanggil handler setting).
func (m *Mailer) Reconfigure(cfg Config) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg = cfg
}

// currentCfg salinan config terkini (aman dipanggil dari goroutine kirim).
func (m *Mailer) currentCfg() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

// New siapkan mailer; log ditulis ke tabel mail_log.
func New(db *sql.DB, cfg Config) *Mailer {
	return &Mailer{db: db, cfg: cfg}
}

// Enabled true bila SMTP dikonfigurasi.
func (m *Mailer) Enabled() bool { return m != nil && m.currentCfg().Host != "" }

// Send kirim email HTML; selalu catat hasil ke mail_log (juga saat SMTP mati).
func (m *Mailer) Send(to, subject, html string) {
	if m == nil {
		return
	}
	cfg := m.currentCfg()
	if cfg.Host == "" {
		m.log(to, subject, "error", "smtp belum dikonfigurasi")
		return
	}
	err := m.deliver(cfg, to, subject, html)
	if err != nil {
		m.log(to, subject, "error", err.Error())
		return
	}
	m.log(to, subject, "sent", "")
}

func (m *Mailer) deliver(cfg Config, to, subject, html string) error {
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	c, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: cfg.Host}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if cfg.User != "" {
		if err := c.Auth(smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)); err != nil {
			return fmt.Errorf("auth smtp: %w", err)
		}
	}
	from := cfg.From
	if from == "" {
		from = cfg.User
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}
	wr, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := wr.Write(buildMessage(from, to, subject, html)); err != nil {
		return err
	}
	if err := wr.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// TestSend kirim email uji memakai config yang diberikan (tanpa mengubah
// config aktif); hasil tetap dicatat ke mail_log.
func (m *Mailer) TestSend(cfg Config, to, subject, html string) error {
	if m == nil || cfg.Host == "" {
		return fmt.Errorf("smtp belum dikonfigurasi")
	}
	err := m.deliver(cfg, to, subject, html)
	if err != nil {
		m.log(to, subject, "error", err.Error())
		return err
	}
	m.log(to, subject, "sent", "")
	return nil
}

// buildMessage pesan RFC822 minimal, HTML utf-8.
func buildMessage(from, to, subject, html string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=utf-8\r\n\r\n%s",
		from, to, mime.QEncoding.Encode("utf-8", subject), time.Now().UTC().Format(time.RFC1123Z), html)
	return []byte(b.String())
}

func (m *Mailer) log(to, subject, status, errMsg string) {
	if m == nil || m.db == nil {
		return
	}
	m.db.Exec(`INSERT INTO mail_log (to_email, subject, status, error) VALUES (?,?,?,?)`,
		to, subject, status, truncate(errMsg, 500))
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// Style amplop email table-based: semua spacing lewat td padding + bgcolor
// attribute (bukan CSS background/margin/flex) agar kompatibel dengan klien
// email lama (Outlook, webmail). Struktur: header brand, body, footer.
func Style(appName, body string) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="id"><head><meta charset="utf-8"/>` +
		`<meta name="viewport" content="width=device-width,initial-scale=1"/></head>` +
		`<body style="margin:0;padding:0;background-color:#f4f4f5;">` +
		`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#f4f4f5"><tr><td align="center" style="padding:24px 12px;">` +
		`<table role="presentation" width="560" cellpadding="0" cellspacing="0" border="0" bgcolor="#ffffff" style="width:560px;max-width:560px;border:1px solid #e4e4e7;">` +
		`<tr><td bgcolor="#18181b" style="padding:20px 32px;font-family:Arial,Helvetica,sans-serif;">` +
		`<span style="color:#ffffff;font-size:18px;font-weight:bold;">` + appName + `</span>` +
		`<br><span style="color:#a1a1aa;font-size:12px;">Transaksi produk digital</span>` +
		`</td></tr>` +
		`<tr><td style="padding:28px 32px;font-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:1.6;color:#18181b;">` +
		body +
		`</td></tr>` +
		`<tr><td style="padding:16px 32px;border-top:1px solid #e4e4e7;font-family:Arial,Helvetica,sans-serif;font-size:12px;color:#a1a1aa;">` +
		`Email otomatis - jangan dibalas langsung.` +
		`</td></tr></table></td></tr></table></body></html>`)
	return b.String()
}

// Button CTA berbasis tabel (tanpa display:inline-block pada tr/td).
func Button(href, label string) string {
	return `<table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr>` +
		`<td bgcolor="#277758" style="border-radius:8px;">` +
		`<a href="` + href + `" style="display:inline-block;padding:10px 18px;font-family:Arial,Helvetica,sans-serif;font-size:14px;font-weight:bold;color:#ffffff;text-decoration:none;border-radius:8px;">` + label + `</a>` +
		`</td></tr></table>`
}
