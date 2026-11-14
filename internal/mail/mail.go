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

// Mailer kirim email + catat hasil ke mail_log.
type Mailer struct {
	db *sql.DB
	cfg Config
}

// New siapkan mailer; log ditulis ke tabel mail_log.
func New(db *sql.DB, cfg Config) *Mailer {
	return &Mailer{db: db, cfg: cfg}
}

// Enabled true bila SMTP dikonfigurasi.
func (m *Mailer) Enabled() bool { return m != nil && m.cfg.Host != "" }

// Send kirim email HTML; selalu catat hasil ke mail_log (juga saat SMTP mati).
func (m *Mailer) Send(to, subject, html string) {
	if m == nil || m.cfg.Host == "" {
		m.log(to, subject, "error", "smtp belum dikonfigurasi")
		return
	}
	err := m.deliver(to, subject, html)
	if err != nil {
		m.log(to, subject, "error", err.Error())
		return
	}
	m.log(to, subject, "sent", "")
}

func (m *Mailer) deliver(to, subject, html string) error {
	addr := net.JoinHostPort(m.cfg.Host, m.cfg.Port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: m.cfg.Host}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if m.cfg.User != "" {
		if err := c.Auth(smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, m.cfg.Host)); err != nil {
			return fmt.Errorf("auth smtp: %w", err)
		}
	}
	from := m.cfg.From
	if from == "" {
		from = m.cfg.User
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

// Style dasar email HTML (inline agar aman di klien email).
func Style(appName, body string) string {
	return `<!DOCTYPE html><html><body style="margin:0;padding:24px;background:#f4f4f5;font-family:-apple-system,Segoe UI,Roboto,sans-serif;color:#18181b;">
<div style="max-width:560px;margin:0 auto;background:#ffffff;border-radius:12px;padding:32px;border:1px solid #e4e4e7;">
<div style="font-size:18px;font-weight:700;margin-bottom:4px;">` + appName + `</div>
<div style="color:#71717a;font-size:13px;margin-bottom:24px;">Transaksi produk digital</div>` + body + `
<div style="color:#a1a1aa;font-size:12px;margin-top:32px;border-top:1px solid #e4e4e7;padding-top:16px;">Email otomatis — jangan dibalas langsung.</div>
</div></body></html>`
}

// Button CTA inline.
func Button(href, label string) string {
	return `<a href="` + href + `" style="display:inline-block;background:#18181b;color:#ffffff;text-decoration:none;padding:10px 18px;border-radius:8px;font-size:14px;font-weight:600;">` + label + `</a>`
}
