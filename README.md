# go-self-digital-store

Online store **single-tenant** untuk jualan produk digital — Go + SQLite,
pembayaran via [paygateway](https://github.com/jhonoryza/paygateway) (12 gateway),
produk terkirim **otomatis ke email** setelah pembayaran terverifikasi.

Frontend: Inertia (React 19 + Vite 8 + Tailwind v4, build dengan bun) di
Cloudflare Pages; backend: satu binary Go di VPS Docker.

## Fitur

- **Katalog**: landing, daftar & detail produk; 3 jenis produk — berkas unduhan,
  kunci lisensi (stok diisi admin, dikirim 1/order), tautan eksternal
- **Keranjang & checkout**: cart localStorage (harga divalidasi ulang server),
  kupon diskon, guest checkout (email wajib), order terikat akun bila login
- **Pembayaran**: halaman bayar ber-token (anti-enumerasi), webhook
  `/callback/payment` fail-closed per provider, pelunasan idempoten + validasi
  nominal (pembayaran parsial VA ditolak), mode simulasi untuk dev
- **Pengiriman**: email otomatis berisi tautan unduh ber-token / kunci / tautan,
  drip reminder order pending >1 jam, mail log semua upaya kirim
- **Akun user**: dashboard, riwayat pesanan, invoice cetak, tiket bantuan
  (percakapan non-realtime + polling ringan), profil
- **Konsol admin**: statistik, users, orders (tandai lunas manual, kirim ulang
  email), produk CRUD + upload berkas + stok kunci, tiket management, konfigurasi
  gateway runtime (secret tersamar)

## Struktur

```
cmd/store/            binary: serve | migrate | seed
internal/
  config/             env STORE_*
  db/                 SQLite (modernc, pure-Go) + migrasi embedded + backup
  auth/               argon2id, sesi sha256, CSRF turunan sesi, lockout
  mail/               SMTP Brevo (STARTTLS) + mail_log
  web/                gonertia v3 + routes + middleware (CORS, rate limit)
    admin.go          konsol admin
    checkout.go       checkout, halaman bayar, webhook, markOrderPaid
    delivery.go       kunci lisensi, email produk, unduhan ber-token
web/                  Inertia (bun + Vite + React + Tailwind v4)
  src/api.ts          klien split-origin: prefix API_BASE, credentials, CSRF
  src/pages/          Public/ Auth/ Account/ Admin/
Dockerfile            backend + frontend fallback dalam satu image
```

## Menjalankan lokal (dev)

```bash
# terminal 1 — backend
make run                     # Go di :8080 (migrasi otomatis)

# terminal 2 — frontend (vite dev, proxy ke :8080)
cd web && bun install && bun run dev
# buka http://localhost:5173 — mode simulasi aktif (provider mock)
```

Seed admin + produk contoh: `make seed` (admin@example.com / admin12345 — ganti!).

Keranjang → checkout → halaman bayar → tombol "Tandai lunas (simulasi)" →
produk tampil + email delivery tercatat di `/admin/maillog`.

## Verifikasi

```bash
make test    # go vet + go test + typecheck + build frontend
```

Test integrasi mencakup: auth (register/login/reset), checkout → webhook
ber-signature (fake gateway httptest) → pelunasan → kunci ter-assign → unduhan
ber-token → mail_log, tiket, dan guard admin.

## Deploy

Lihat [DEPLOY.md](DEPLOY.md) — VPS Docker + Cloudflare Pages + Brevo SMTP +
konfigurasi gateway, termasuk catatan protokol Inertia split-origin.

## Lisensi

[MIT](LICENSE)
