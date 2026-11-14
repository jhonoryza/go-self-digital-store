# AGENTS.md — go-self-digital-store

Toko online single-tenant untuk jualan produk digital. Pembayaran via library
`github.com/jhonoryza/paygateway` (12 gateway + simulasi); setelah lunas, produk
terkirim otomatis ke email pembeli (unduhan ber-token / kunci lisensi / tautan).

## Stack & arsitektur

- **Backend**: Go 1.26, SQLite pure-Go (`modernc.org/sqlite`), Inertia server
  `romsar/gonertia/v3`, SMTP Brevo (stdlib), nol framework web — `net/http` mux.
- **Frontend**: Inertia (React 19 + Vite 8 Rolldown + Tailwind v4), package
  manager **bun**, direktori `web/`.
- **Deploy split-origin**: shell statis di Cloudflare Pages, API Go di VPS
  Docker. Kontrak lintas-origin dipegang dua file: `web/src/api.ts` (klien) dan
  `internal/web/inertiamw.go` (server) — baca keduanya sebelum menyentuh
  protokol Inertia. Detail: DEPLOY.md §5.
- **Gambar produk** tersimpan di bucket Cloudflare R2 bersama `webapp`
  (prefix per-app `go-self-digital-store/`); kredensial dari `STORE_R2_*`
  atau fallback `~/.mc/config.json` — lihat internal/web/images.go.
- **Mode ganda**: shell dirender Go (dev/single-origin, aset dari
  `web/dist/.vite/manifest.json`) ATAU shell statis CF Pages (fetch page JSON).
  Keduanya mem-boot app React yang sama.

## Perintah

| Aksi | Perintah |
|---|---|
| Jalankan lokal | `make run` (Go :8080; login-as dev + SMTP Mailpit aktif) + `cd web && bun run dev` (:5173, proxy ke :8080) |
| Verifikasi lengkap | `make test` (go vet + go test + typecheck + build web) |
| Build frontend | `cd web && bun run build` → `web/dist/` |
| Migrasi / seed | `./store migrate` / `./store seed` (admin + 4 produk contoh) |
| Docker | `docker compose up -d --build` |

## Konvensi

- Komentar & pesan commit Bahasa Indonesia; pesan commit ber-prefix domain
  (`M4:` / `perbaikan:` / `fitur:`).
- Semua env var prefix `STORE_*` — definisi di `internal/config/config.go`,
  didokumentasikan di DEPLOY.md.
- Nama provider paygateway (`midtrans`, `tripay`, …, `mock`) TIDAK boleh berubah —
  tersimpan sebagai string di settings DB. Adapter baru ditambah di repo
  paygateway, bukan di sini.
- Timestamp DB pakai `datetime('now')` SQLite (TEXT UTC).
- respons JSON handler: `jsonError(w, code, msg)`; flash redirect: `?ok=`/`?err=`
  via `s.flashOK/flashErr`.

## Aturan keras (pernah jadi bug — jangan dilanggar)

1. **SQLite koneksi tunggal** (`SetMaxOpenConns(1)`): JANGAN pernah query
   bersarang saat `rows` masih terbuka — deadlock. Kumpulkan data dulu, tutup
   rows, baru query berikutnya.
2. **argon2id**: format encoded 5 segmen `$` (`argon2id$v=1$...$salt$hash`) —
   lihat internal/auth.
3. **Pelunasan order selalu idempoten**: `UPDATE ... WHERE status!='paid'` +
   cek `RowsAffected()==1`; validasi nominal callback vs total order.
4. **Webhook fail-closed**: secret kosong / header signature hilang = 401.
5. **Frontend build menambah hash aset baru** → restart server Go setelah
   `bun run build` (versi aset dibaca saat startup).

## Child DOX Index

- [cmd/store](cmd/store/AGENTS.md) — binary CLI: serve/migrate/seed
- [internal/db](internal/db/AGENTS.md) — SQLite, migrasi embedded, backup
- [internal/auth](internal/auth/AGENTS.md) — password, sesi, CSRF, lockout
- [internal/mail](internal/mail/AGENTS.md) — SMTP Brevo + mail_log
- [internal/web](internal/web/AGENTS.md) — server, protokol Inertia, semua handler + test
- [web](web/AGENTS.md) — frontend Inertia (bun/Vite/React/Tailwind), kontrak split-origin
