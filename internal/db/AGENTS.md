# AGENTS.md — internal/db

## Purpose

SQLite pure-Go (`modernc.org/sqlite`): Open, migrasi embedded, backup VACUUM.

## Ownership

- `db.go`: DSN pragma (WAL, busy_timeout 5000, synchronous NORMAL, foreign_keys),
  **`SetMaxOpenConns(1)`** — tulis SQLite eksklusif; >1 koneksi menyebabkan
  SQLITE_BUSY senyap.
- `migrations/NNNN_name.sql` — dimuat `//go:embed`, versi dicatat di
  `schema_migrations`, dijalankan dalam transaction per file.

## Local Contracts

- Skema 0001_init.sql: users, sessions, password_resets, settings, products,
  license_keys, coupons, orders, order_items, tickets, ticket_messages,
  mail_log, audit_log. Kolom `orders.dl_token` dibuat saat **checkout**
  (bukan saat lunas) — kunci halaman bayar + unduhan.
- `orders.order_no` (ORD-…) = referensi publik; `id` internal.
- Migration baru: buat file `NNNN Deskripsi.sql` berikutnya, tidak pernah edit
  migration lama yang sudah ter-apply.

## Work Guidance

- **Jangan query bersarang saat rows terbuka** (koneksi tunggal → deadlock).
  Pola aman: scan semua ke slice → `rows.Close()` → query berikutnya.
  Lihat contoh `loadOrder` di internal/web/checkout.go.
- QueryRow().Scan() boleh di antara query (auto-close), tapi tetap hindari
  di dalam loop `rows.Next()` bila memungkinkan.
- `Backup(d, dir, keep)` = `VACUUM INTO` — online copy, dipanggil worker.

## Verification

- `go test ./internal/...` — migrasi dijalankan oleh `testServer` di
  internal/web/server_test.go (SQLite temp-dir nyata).

## Child DOX Index

- `migrations/` — file SQL versi; tanpa AGENTS.md (kontrak di dokumen ini).
