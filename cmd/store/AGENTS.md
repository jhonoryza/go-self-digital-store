# AGENTS.md — cmd/store

## Purpose

Entry point binary `store` — tiga subcommand tanpa flag library eksternal.

## Ownership

- `main.go`: dispatch subcommand, wiring config→db→mail→web, worker latar
  (sweep rate limiter, `srv.Maintain()` reminder, backup berkala).

## Local Contracts

- `serve`: buka DB → migrate otomatis → `web.NewInertia` → `web.New` → listen
  `STORE_ADDR`.
- `migrate`: migrasi lalu keluar (dipakai Docker/compose init).
- `seed`: idempoten — skip bila email admin sudah ada. Default
  `admin@example.com / admin12345` (override: `STORE_SEED_ADMIN_EMAIL/PASSWORD`).
  Seed juga 4 produk contoh (file/key/link) + dummy hingga 50, semuanya dengan
  kategori (prefix nama) dan tags (jenis + pool) — slug di-skip bila sudah ada.
- Worker loop interval 15 menit; backup hanya jalan bila SMTP enabled (proxy
  sederhana "produksi aktif").

## Work Guidance

- Subcommand baru = tambah case di `main()` + tulis kontraknya di sini.
- Jangan taruh logika bisnis di main — semua handler di internal/web.

## Verification

- `go build ./... && go vet ./...`
- Smoke: `./store migrate && ./store seed && ./store serve` → `GET /healthz`
  harus `{"ok":true}`.

## Child DOX Index

(tidak ada — leaf)
