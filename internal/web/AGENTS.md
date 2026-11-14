# AGENTS.md — internal/web

## Purpose

Seluruh HTTP layer: setup gonertia, rantai middleware, semua handler (publik,
auth, akun, checkout/pembayaran, delivery, tiket, admin), dan test integrasi.

## Ownership

Rantai middleware (`New()`): `recover500 → secHeaders → cors → originCheck →
inertiaMiddleware → mux`. Urutan ini kontrak — originCheck sebelum inertia agar
request asing ditolak sebelum di-buffer.

| File | Isi |
|---|---|
| `server.go` | Server struct, `routes()` (satu fungsi flat, method-pattern mux), serveAssets, authOnFail |
| `inertiamw.go` | Middleware Inertia buatan sendiri (bukan `i.Middleware`) — respons **dibuffer penuh** (bufferedResponse) sebelum disalin |
| `inertia.go` | Root template + resolver aset (manifest/hot) + version |
| `middleware.go` | secHeaders, CORS, rate limit in-memory (`ipLimits`), clientIP (proxy-aware), Sweep |
| `helpers.go` | settings, audit, rupiah, truncate, Page() (props bersama: auth_user + csrf_token), Redirect, flash ?ok=/&err= |
| `public.go` | landing, list/detail produk, healthz |
| `auth.go` | register/login/logout, lupa+reset password, lockout (5 gagal→10 menit, timing equalizer) |
| `account.go` `orders.go` | profil, ganti password, pesanan milik sendiri (user_id ATAU email cocok), invoice, dashboard stats |
| `checkout.go` | checkout tx, halaman bayar ber-token, webhook, `markOrderPaid` idempoten + validasi nominal, `loadOrder` |
| `gateway.go` | paygateway: settings `payment_provider`/`payment_cfg.<name>`, `activeProvider` (fallback Mock), `createCharge` |
| `delivery.go` | onOrderPaid (assign kunci RETURNING, email produk), `/dl` ber-token, remindPending, `Maintain()` |
| `tickets.go` | tiket user: list/buat/balas; status open→answered→closed |
| `admin.go` | konsol admin: stats, users, orders (lunas manual/resend), produk CRUD multipart + stok kunci, maillog, tiket admin, gateway config/activate |
| `server_test.go` | test integrasi (bootstrap `testServer`: SQLite temp-dir nyata + httptest) |
| `e2e_test.go` | test E2E per fitur (publik, auth, profil, checkout, webhook, unduhan, tiket, admin, middleware) |
| `dev.go` | dev login-as (`/dev/login`, hanya bila `STORE_DEV_LOGIN=1`; route tidak terdaftar bila mati) |

## Local Contracts

- **inertiaMiddleware**: respons dibuffer; GET Inertia **tanpa** `X-Inertia-Version`
  = fetch awal shell statis → dilayani (toleransi versi kosong); versi ≠ server →
  409 + `X-Inertia-Location` path-relative; kosong 200 → Back; 302 setelah
  PUT/PATCH/DELETE → 303.
- **originCheck**: method tulis dengan header Origin harus cocok host API atau
  CORS allowlist; localhost diizinkan hanya saat CORS kosong (dev via vite proxy).
  Webhook tanpa Origin lolos (non-browser).
- **mergeJSONBody**: klien Inertia v3 kirim form JSON → di-merge ke `r.PostForm`
  sehingga handler pakai `r.PostFormValue` seperti biasa. Hanya untuk request
  Inertia (body mentah webhook tidak tersentuh).
- `s.Page()` selalu menyuntik `auth_user` + `csrf_token` ke props.
- Validasi gagal → `renderWithErrors` (render ulang komponen + `props.errors`)
  — tanpa redirect. Sukses → `s.Redirect` (302; 303 dikonversi middleware).
- Redirect eksternal (URL gateway) → `s.I.Location` (409 + X-Inertia-Location).
- Pelunasan: `markOrderPaid` → `onOrderPaid` HANYA bila RowsAffected==1.
- `publicOrigin(r)` = CORS origin pertama (frontend) fallback BaseURL — dipakai
  untuk semua URL yang dibuka manusia (ReturnURL gateway, tautan email).
  `baseURL(r)` = origin API (webhook, link unduhan di email).
- **Versi Inertia selalu md5**: gonertia menyimpan `md5(WithVersion(arg))` — JSON
  halaman mengirim hasil md5 itu. `NewInertia` mengembalikan versi yang SUDAH
  di-md5 (`inertiaVersion`) supaya `s.Version` (dipakai inertiamw membandingkan
  header `X-Inertia-Version`) identik dengan yang dilihat klien. Jangan
  membandingkan versi mentah — semua GET ber-version akan 409 selamanya.
- Route GET `/keranjang` & `/checkout` wajib ada di `routes()` meski halamannya
  murni klien (isi keranjang di localStorage) — navbar & flash checkout
  mengarah ke sana; di mode shell Go tanpa route = 404.

## Work Guidance

- Handler baru: daftarkan di `routes()` + tulis di file domain; guard via
  `s.guardAuth` / `s.guardAdmin`; rate limit via `s.limited("kind", …)`.
- Kuota rate limit per endpoint ada di `ipLimits` — tambah kind baru di situ.
- Admin aksi selalu `audit(s.DB, action, target, detail)`.
- Jangan panggil gonertia `i.Middleware` — versi buffer milik sendiri.
- Test baru ikuti pola `server_test.go`: `testServer`, `newClient()` (cookie jar
  + no-follow), `doPost` (JSON + X-Inertia + Origin), `doPage` (parse Inertia
  JSON), `csrfOf`. Rate limiter global direset di `testServer`.

## Verification

- `go test ./internal/web/ -race` — dua kelompok:
  - **server_test.go** (6 suite): auth/lockout/reset, checkout→webhook tripay
    ber-signature→delivery→unduhan, tiket, admin guard.
  - **e2e_test.go** (11 suite): halaman publik, auth validasi+guard sesi+logout,
    profil+ganti password (sesi lain putus), dashboard/pesanan/invoice,
    validasi checkout+kupon, pembayaran simulasi (payNow/stok habis/idempoten),
    webhook tripay negatif (EXPIRED/nominal beda/order tak dikenal/fail-closed),
    guard unduhan /dl, siklus tiket penuh (closed, akses silang), konsol admin
    (produk multipart CRUD, stok kunci, toggle, order manual/resend, gateway
    masked), middleware keamanan (secHeaders/origin/CSRF/rate limit/versi
    Inertia/CORS), plus alur lintas fitur beli→tiket→invoice.
- `make test` = vet + test + typecheck + build web.

## Child DOX Index

(tidak ada — semua handler di satu package; pecah AGENTS.md anak bila internal/web
menjadi terlalu besar)
