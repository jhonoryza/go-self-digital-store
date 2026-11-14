// Lapisan transport Inertia untuk deploy split-origin (shell statis CF Pages
// + API Go di VPS):
//   1. URL relatif diprefiks API_BASE (VITE_API_BASE; kosong = same-origin/dev).
//   2. Klien XHR di-subclass agar withCredentials=true — cookie sesi SameSite=None
//      terkirim pada XHR lintas origin.
//   3. Interceptor menempel X-CSRF-Token (turunan sesi, dikirim lewat props).
//   4. fetchInitialPage mengambil page JSON tanpa X-Inertia-Version — server
//      menoleransi versi kosong untuk pemuatan awal shell (lihat internal/web/inertiamw.go).

import { http, XhrHttpClient } from "@inertiajs/core";
import type { HttpRequestConfig, HttpResponse, Page, PageProps } from "@inertiajs/core";

export const API_BASE = import.meta.env.VITE_API_BASE ?? "";

let csrf = "";
export function setCsrf(token: string) {
  csrf = token ?? "";
}

function isAbsolute(url: string) {
  return /^https?:\/\//i.test(url) || url.startsWith("//");
}

function absoluteUrl(url: string) {
  if (!API_BASE || !url.startsWith("/") || isAbsolute(url)) return url;
  return API_BASE + url;
}

// Salin XhrHttpClient.doRequest dengan satu tambahan: withCredentials = true.
// (Klien bawaan tidak mengirim cookie pada XHR lintas origin.)
class CredentialsClient extends XhrHttpClient {
  doRequest(config: HttpRequestConfig): Promise<HttpResponse> {
    return new Promise((resolve, reject) => {
      if (config.signal?.aborted) {
        reject(new Error("cancelled"));
        return;
      }
      const xhr = new XMLHttpRequest();
      const url = absoluteUrl(config.url);
      xhr.open(config.method.toUpperCase(), url, true);
      xhr.withCredentials = true; // <- satu-satunya perubahan dari klien bawaan
      const hasRequestedWithHeader = Object.keys(config.headers ?? {}).some(
        (key) => key.toLowerCase() === "x-requested-with",
      );
      if (!hasRequestedWithHeader) {
        xhr.setRequestHeader("X-Requested-With", "XMLHttpRequest");
      }
      let body: XMLHttpRequestBodyInit | null = null;
      if (config.data !== null && config.data !== undefined) {
        if (
          config.data instanceof FormData ||
          config.data instanceof Blob ||
          config.data instanceof ArrayBuffer ||
          config.data instanceof URLSearchParams ||
          typeof config.data === "string"
        ) {
          // body mentah (FormData untuk upload berkas) — kirim apa adanya
          body = config.data;
        } else if (typeof config.data === "object") {
          body = JSON.stringify(config.data);
          if (!config.headers?.["Content-Type"] && !config.headers?.["content-type"]) {
            xhr.setRequestHeader("Content-Type", "application/json");
          }
        } else {
          body = String(config.data);
        }
      }
      for (const [k, v] of Object.entries(config.headers ?? {})) {
        xhr.setRequestHeader(k, String(v));
      }
      if (config.signal) {
        config.signal.addEventListener("abort", () => xhr.abort());
      }
      xhr.onabort = () => reject(new Error("cancelled"));
      xhr.onerror = () => reject(new TypeError("network error"));
      xhr.onload = () => {
        const headers: Record<string, string> = {};
        xhr.getAllResponseHeaders()
          .split("\r\n")
          .forEach((line) => {
            const idx = line.indexOf(":");
            if (idx > 0) headers[line.slice(0, idx).trim().toLowerCase()] = line.slice(idx + 1).trim();
          });
        const response: HttpResponse = { status: xhr.status, data: xhr.responseText, headers };
        if (xhr.status >= 400) {
          const err = new Error(`Request failed with status ${xhr.status}`) as Error & {
            response?: HttpResponse;
          };
          err.response = response;
          reject(err);
        } else {
          resolve(response);
        }
      };
      xhr.send(body);
    });
  }
}

http.setClient(new CredentialsClient());

// Prefix origin API + header CSRF untuk method tulis.
http.onRequest((cfg) => {
  const url = absoluteUrl(cfg.url);
  const method = cfg.method.toUpperCase();
  const headers = { ...(cfg.headers ?? {}) };
  // Inertia v3 mengirim cfg.url sudah absolut (resolved terhadap origin) walau
  // same-origin — jangan gate pada !isAbsolute, semua visit tulis ke API kita
  // wajib membawa X-CSRF-Token (turunan sesi API) agar Guard server lolos.
  if (method !== "GET" && method !== "HEAD" && csrf) {
    headers["X-CSRF-Token"] = csrf;
  }
  return { ...cfg, url, headers };
});

// Halaman awal untuk shell statis: fetch page JSON dari API.
// 409 + X-Inertia-Location (auth/version) → ikuti lokasi (relative = shell origin).
export async function fetchInitialPage(path: string): Promise<Page<PageProps>> {
  const res = await fetch(absoluteUrl(path), {
    headers: {
      "X-Inertia": "true",
      "X-Requested-With": "XMLHttpRequest",
      Accept: "text/html, application/xhtml+xml",
    },
    credentials: "include",
  });
  const loc = res.headers.get("X-Inertia-Location");
  if (loc) {
    window.location.href = loc;
    return new Promise(() => {
      /* navigasi browser mengambil alih */
    });
  }
  try {
    return await res.json();
  } catch {
    // respons bukan Inertia (mis. HTML lama) — muat ulang shell
    window.location.reload();
    return new Promise(() => {
      /* tunggu reload */
    });
  }
}
