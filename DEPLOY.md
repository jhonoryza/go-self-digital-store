# Deploy

Arsitektur produksi: **frontend statis di Cloudflare Pages, API Go di VPS dengan
Docker**. Inertia split-origin — lihat catatan protokol di bagian bawah.

## 1. Backend di VPS (Docker)

```bash
# di server
git clone https://github.com/jhonoryza/go-self-digital-store && cd go-self-digital-store
cp .env.example .env  # isi (lihat di bawah)
docker compose up -d --build
./store migrate && ./store seed   # atau: docker compose exec store ./store seed
```

Backend hanya listen di `127.0.0.1:8080` — taruh reverse proxy TLS di depan.
Contoh Caddyfile:

```
api.tokokamu.com {
    reverse_proxy 127.0.0.1:8080
}
```

atau Nginx:

```nginx
server {
    server_name api.tokokamu.com;
    listen 443 ssl http2;
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

`STORE_TRUSTED_PROXY=1` wajib bila di belakang proxy agar rate-limit per-IP akurat.

### Env vars (prefix `STORE_*`)

| Var | Arti |
|---|---|
| `STORE_ADDR` | listen address (default `:8080`) |
| `STORE_DIR` | direktori data: `store.db`, `files/`, `backups/` |
| `STORE_NAME` | nama toko (judul halaman, email) |
| `STORE_BASE_URL` | URL publik API, mis. `https://api.tokokamu.com` — tautan di email |
| `STORE_CORS_ORIGINS` | origin frontend CF Pages (koma, mis. `https://tokokamu.pages.dev`) |
| `STORE_SMTP_HOST/PORT/USER/PASS/FROM` | SMTP Brevo: `smtp-relay.brevo.com:587`, user = login SMTP, pass = **SMTP key** |
| `STORE_PAYMENT_SECRET` | HMAC webhook adapter kustom (opsional) |
| `STORE_COOKIE_SECURE` | force cookie Secure (otomatis bila BASE_URL https) |
| `STORE_COOKIE_SAMESITE` | `none` untuk split-origin (otomatis bila CORS origins diisi), `lax` same-origin |
| `STORE_TRUSTED_PROXY` | `1` bila di belakang reverse proxy |
| `STORE_BACKUP_KEEP` | jumlah backup SQLite disimpan (default 7) |

## 2. Frontend di Cloudflare Pages

1. Pages → buat project → **Connect to Git** pilih repo ini.
2. Build settings:
   - Build command: `cd web && bun install && bun run build`
   - Output directory: `web/dist`
3. Environment variable (production):
   - `VITE_API_BASE` = `https://api.tokokamu.com`
4. SPA fallback & cache sudah disertakan: `web/public/_redirects` (`/* /index.html 200`)
   dan `web/public/_headers` (`/assets/*` immutable).

Cara kerja: shell statis (`index.html`) mem-boot React → fetch page JSON pertama
dari `VITE_API_BASE` (header `X-Inertia`, tanpa versi) → navigasi lanjutan lewat
XHR ber-credentials ke API. Backend mengizinkan origin Pages via `STORE_CORS_ORIGINS`.

## 3. Brevo SMTP

Buat SMTP key di app.brevo.com/settings/keys/smtp (bukan API key). Isi
`STORE_SMTP_HOST=smtp-relay.brevo.com`, `PORT=587`, `USER=<login smtp>`,
`PASS=<smtp key>`, `FROM=<pengirim terverifikasi>`. Free plan: 300 email/hari.
Tanpa SMTP, app tetap jalan (upaya email tercatat di mail_log sebagai error).

## 4. Gateway pembayaran

Semua konfigurasi runtime di konsol `/admin/gateway` (login admin): pilih provider,
isi kredensial per field (secret tersamar), klik Aktifkan. Nama provider stabil
(midtrans, tripay, xendit, doku, ipaymu, espay, singapay, mayar, sumopod,
lemonsqueezy, creem, polar, mock) — tersimpan di settings table.

Set webhook gateway ke `STORE_BASE_URL + /callback/payment` di dashboard
masing-masing gateway. Untuk pengujian lokal/dev gunakan provider `mock`
(simulasi) — tombol tandai-lunas muncul di halaman bayar.

## 5. Protokol split-origin (catatan teknis)

- Cookie sesi `SameSite=None; Secure` — wajib HTTPS di kedua origin.
- CSRF: token turunan sesi dikirim lewat props → header `X-CSRF-Token` oleh
  interceptor (api.ts); lapis kedua: origin-check pada semua method tulis.
- Pemeriksaan versi aset Inertia: request **tanpa** `X-Inertia-Version`
  (fetch awal shell) dilayani; versi dikirim ≠ server → 409 + `X-Inertia-Location`
  path-relative (klien memuat ulang shell, bukan API).
- Header CORS yang di-expose: `X-Inertia`, `X-Inertia-Location`, `X-Inertia-Version`.

## 6. Backup

Worker latar melakukan `VACUUM INTO` tiap 15 menit (interval tick) ke
`STORE_DIR/backups/` dengan retensi `STORE_BACKUP_KEEP`. File produk di
`STORE_DIR/files/` — ikutkan juga dalam backup eksternal bila berkas besar.
