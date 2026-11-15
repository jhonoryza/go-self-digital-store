# AGENTS.md — web (frontend Inertia)

## Purpose

Frontend Inertia: React 19 + Vite 8 (Rolldown) + Tailwind v4, package manager
bun. Deploy: shell statis + aset di Cloudflare Pages; lokal: `bun run dev`
(:5173, proxy semua path non-vite ke Go :8080).

## Ownership

- `src/api.ts` — **kontrak split-origin** (paling krusial):
  1. `XhrHttpClient` di-subclass, `withCredentials = true` (cookie SameSite=None
     lintas origin),
  2. `http.onRequest`: prefix `API_BASE` (env `VITE_API_BASE`) untuk URL relatif +
     header `X-CSRF-Token` (dari props) pada method tulis,
  3. `fetchInitialPage`: fetch page JSON dari API **tanpa** `X-Inertia-Version`
     (server menoleransi versi kosong), ikuti `X-Inertia-Location` bila 409.
- `src/app.tsx` — bootstrap: `fetchInitialPage` → `createInertiaApp({ page })`;
  resolver `import.meta.glob('./pages/**/*.tsx')`; **layout persisten diapply
  manual** di callback children (callback kustom menonaktifkan layout bawaan
  App); `CsrfSync` menyinkronkan token CSRF props → interceptor.
- `src/types.ts` — SharedProps + helper `rupiah` (cermin helper Go).
- `src/components/ThemeToggle.tsx` — toggle gelap/terang (localStorage
  `theme`); **default gelap** diputuskan script inline di shell (Go
  `rootTemplate` + `web/index.html`) sebelum paint.
- `src/components/ProductTypeIcon.tsx` — ikon jenis produk (Package/KeyRound/
  Link2) dari lucide-react; satu sumber untuk seluruh katalog.
- `src/pages/Dev/LoginAs.tsx` — masuk sebagai akun mana pun (hanya bila
  prop `dev_login` aktif; route Go tidak terdaftar tanpa `STORE_DEV_LOGIN=1`).
- `src/lib/cart.ts` — keranjang localStorage (event `cart-change`); server
  memvalidasi ulang harga saat checkout. `CartItem.image_url` snapshot gambar.
- `src/components/ProductThumb.tsx` — kartu gambar produk (foto pertama dari
  R2, fallback ikon jenis); dipakai landing/katalog/keranjang.
- `src/pages/Admin/Products.tsx` — `ImageManager` unggah/hapus gambar (maks 5,
  jpg/png/webp, 2 MB) via multipart `images` ke `/admin/produk/{id}/images`;
  unggah butuh produk sudah tersimpan (ada id).
- `src/pages/{Public,Auth,Account,Admin,Dev}` — halaman; nama komponen = path
  komponen Inertia dari backend (mis. `Public/Landing`).
- `src/pages/Admin/Settings.tsx` — setting runtime (SMTP via Mailer.Reconfigure,
  brand nama+logo di R2, konten privasi/ketentuan, SEO per halaman); semua
  tersimpan di tabel settings. `Admin/OrderDetail.tsx` — detail order versi
  admin (kunci lisensi + uji tautan unduh + aksi lunas/resend).
- **SEO/meta**: gunakan `<Head>` dari @inertiajs/react. Sumber: settings
  `seo.*` (halaman setting admin) untuk beranda/katalog; produk punya
  meta_title/meta_description sendiri (form admin, kolom migrasi 0003);
  Policy memakai konten settings bila terisi (fallback: array bawaan).
- `public/_redirects` (`/* /index.html 200`) + `public/_headers` (`/assets/*`
  immutable) — wajib untuk CF Pages.

## Local Contracts

- **Inertia v3**: field form via `form.data.X` (bukan `form.X`);
  `useForm` kirim JSON (FormData bila ada File — klien kami meneruskan mentah);
  callback children App menerima satu objek `{Component, props, key}`.
- **CSS wajib di-import dari entry** (`import "./css/app.css"` di app.tsx) —
  tanpa itu Vite 8 tidak meng-emit CSS sama sekali.
- Font: **Ubuntu** (sans) + **Ubuntu Mono**, di-host sendiri via `@fontsource/ubuntu`
  + `@fontsource/ubuntu-mono` (import bobot 400/500/700 di `app.tsx`; tanpa
  `<link>` Google Fonts). Token `--font-sans` / `--font-mono` di `app.css`.
- Tailwind v4 CSS-first: `@plugin "@tailwindcss/forms"`, `@custom-variant dark`
  (strategy class; `<html>` diberi `class="dark"` default), tema brand
  (`brand` hijau pinus, `brass` kuningan) di `src/css/app.css`, plus rule
  `:focus-visible` global.
- Ikon: **lucide-react satu keluarga untuk seluruh app** — larangan emoji
  sebagai ikon dan SVG hand-rolled; `strokeWidth` bawaan (2).
- Copy user-visible **tanpa em-dash (`—`)** — pakai hyphen atau pisah kalimat
  (berlaku juga untuk pesan flash/email di sisi Go).
- Halaman baru: buat file di `src/pages/<Domain>/<Nama>.tsx`, sematkan layout
  (`Nama.layout = (p) => <Layout>{p}</Layout>`), tambahkan route + handler Go
  (resolve dari glob otomatis).
- `bun run typecheck` harus bersih — semua props halaman bertipe (interface
  extends `SharedProps`).

## Work Guidance

- Jangan ubah `src/api.ts` tanpa memahami `internal/web/inertiamw.go` — keduanya
  satu kontrak protokol.
- Setelah `bun run build`, restart server Go (versi aset dibaca saat startup).
- Copy UI Bahasa Indonesia; halaman cetak (invoice) sertakan `print:hidden`
  untuk elemen non-cetak. **Cetak dipaksa tinta gelap di kertas walau UI
  gelap** — aturan `@media print` di `src/css/app.css` (`.dark *` dinetralkan,
  `@page` A4); header AccountLayout `print:hidden`; tombol unduh invoice
  `print:hidden` (yang tercetak: kunci lisensi & teks).

## Verification

- `bun run typecheck && bun run build` (bagian dari `make test`).
- Visual: Chrome headless screenshot `http://localhost:8080/...` setelah build +
  restart Go (lihat riwayat — halaman putih = aset/versi stale).

## Child DOX Index

- `src/pages/` — halaman per domain (Public/Auth/Account/Admin); tanpa AGENTS.md
  sendiri, kontrak di dokumen ini.
