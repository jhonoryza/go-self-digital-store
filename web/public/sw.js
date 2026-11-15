// Service worker toko digital — strategi:
//   - navigasi (HTML/JSON Inertia): network-first, fallback offline.html
//   - aset statis (assets/, icons/, gambar R2): stale-while-revalidate
//   - JANGAN pernah: /dl/ (presigned), /pay/, /admin, /api, non-GET
// Versi naikkan saat manifest/sw berubah agar cache lama terhapus.
const VERSION = "v1";
const OFFLINE_URL = "/offline.html";

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(VERSION).then((cache) => cache.add(OFFLINE_URL)).then(() => self.skipWaiting()),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== VERSION).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  const req = event.request;
  if (req.method !== "GET") return;
  const url = new URL(req.url);
  if (url.origin !== location.origin) return;
  // jalur sensitif / dinamis: jangan disentuh service worker
  if (
    url.pathname.startsWith("/dl/") ||
    url.pathname.startsWith("/pay/") ||
    url.pathname.startsWith("/admin") ||
    url.pathname.startsWith("/api") ||
    url.pathname.startsWith("/dev")
  ) {
    return;
  }

  // navigasi halaman: coba jaringan dulu, gagal → offline
  if (req.mode === "navigate") {
    event.respondWith(
      fetch(req).catch(() =>
        caches.match(OFFLINE_URL).then((r) => r || Response.error()),
      ),
    );
    return;
  }

  // aset: stale-while-revalidate
  const cacheable =
    url.pathname.startsWith("/assets/") ||
    url.pathname.startsWith("/icons/") ||
    url.pathname.startsWith("/images/") ||
    /\.(png|jpg|jpeg|webp|svg|ico|css|js|woff2?|webmanifest)$/.test(url.pathname);
  if (!cacheable) return;

  event.respondWith(
    caches.open(VERSION).then(async (cache) => {
      const cached = await cache.match(req);
      const jaringan = fetch(req)
        .then((res) => {
          if (res.ok) cache.put(req, res.clone());
          return res;
        })
        .catch(() => cached || Response.error());
      return cached || jaringan;
    }),
  );
});
