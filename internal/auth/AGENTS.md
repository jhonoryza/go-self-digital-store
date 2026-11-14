# AGENTS.md — internal/auth

## Purpose

Autentikasi tanpa dependensi: argon2id, sesi cookie, CSRF stateless, lockout.

## Ownership

- `password.go`: HashPassword/VerifyPassword argon2id (m=64MB, t=1, p=4).
- `session.go`: sesi + CSRF + Guard middleware + cookie config.

## Local Contracts

- **Format hash argon2id = 5 segmen `$`** (`argon2id$v=1$m=..,t=..,p=..$salt$hash`).
  VerifyPassword menolak selain 5 — pernah jadi bug login total.
- Sesi: cookie `store_session` berisi token mentah; DB simpan `sha256(token)`;
  TTL 30 hari; expired diprune saat StartSession.
- **CSRF stateless**: `sha256(token_sesi + ":csrf")` — dikirim ke klien via
  props Inertia (`csrf_token`), dikembalikan lewat header `X-CSRF-Token` oleh
  interceptor `web/src/api.ts`. Bukan cookie ganda — tetap bekerja lintas origin.
- Cookie: HttpOnly; `Secure` + `SameSite=None` saat split-origin (config),
  `SameSite=Lax` same-origin. None tanpa Secure ditolak browser.
- `Guard(db, cfg, requireAdmin, next, onFail)`: isi context user, cek CSRF pada
  POST/PUT/PATCH/DELETE, admin gate role. `onFail` menentukan bentuk respons
  (409+X-Inertia-Location untuk XHR Inertia, 302 untuk biasa) — ada di
  internal/web/server.go `authOnFail`.
- Login lockout in-memory (middleware web, bukan di sini): 5 gagal → 10 menit,
  key `email|IP`.

## Work Guidance

- Jangan pernah mengirim token sesi mentah ke klien selain cookie.
- User di context: `auth.FromContext(ctx)`; resolve manual: `auth.Resolve`.
- Ganti password wajib `DeleteOtherSessions` (sesi lain putus, sesi ini tetap).

## Verification

- `go test ./internal/web/ -race` — server_test.go (auth/lockout/reset) dan
  e2e_test.go (validasi register/login, guard sesi, ganti password memutus
  sesi lain) mencakup kontrak ini.

## Child DOX Index

(tidak ada — leaf)
