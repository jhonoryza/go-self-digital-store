# AGENTS.md — internal/mail

## Purpose

Kirim email via SMTP relay (Brevo `smtp-relay.brevo.com:587` STARTTLS oportunistik,
stdlib `net/smtp`) + catat **setiap upaya** ke tabel `mail_log`.

## Ownership

- `mail.go`: Mailer (db + config), Send/deliver, buildMessage RFC822 HTML utf-8,
  Style(appName, body) template email inline-style, Button(href, label).

## Local Contracts

- `Send(to, subject, html)` selalu menulis mail_log: `sent` / `error` (error
  berisi pesan singkat, termasuk "smtp belum dikonfigurasi" saat dev).
- `Enabled()` = SMTP host terisi; app tetap jalan tanpa email.
- Kirim dari handler selalu dalam goroutine (async) — mail_log ditulis oleh
  Mailer sendiri.
- Kredensial dari `STORE_SMTP_*`; USER = login SMTP Brevo, PASS = **SMTP key**
  (bukan API key).

## Work Guidance

- Email baru = susun body dengan `mail.Style` + `mail.Button`, panggil
  `s.Mail.Send` dalam `go func()`. Jangan log isi pesan — cukup to/subject.
- Kredensial tidak pernah masuk log/commit.

## Verification

- Test integrasi memeriksa mail_log berisi baris email delivery (status error
  saat SMTP mati) — server_test.go TestCheckoutAndTripayWebhook.

## Child DOX Index

(tidak ada — leaf)
