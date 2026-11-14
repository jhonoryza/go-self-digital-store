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
- Uji email lokal pakai **Mailpit** (sudah terpasang; SMTP 127.0.0.1:1025, UI
  http://localhost:8025) — `make run` sudah mengarah ke sana.

## Verification

- e2e: `waitMailLog` di internal/web/images_test.go/e2e_test.go memeriksa
  mail_log terisi (status `error` saat SMTP mati).
- Manual: Mailpit menampilkan email reset password + delivery (tautan
  `/dl/...` ber-token) — status `sent` di mail_log.

## Child DOX Index

(tidak ada — leaf)
