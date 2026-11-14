package web

// Middleware Inertia (adaptasi gonertia i.Middleware) dengan satu penyesuaian
// untuk deploy split-origin (shell statis CF Pages + API VPS): permintaan GET
// Inertia TANPA header X-Inertia-Version dianggap pemuatan awal shell statis —
// dilayani JSON tanpa 409. Versi dikirim ≠ versi server → 409 + X-Inertia-Location
// path-relative (klien memuat ulang shell di origin-nya sendiri, bukan API).
// Respons dibuffer penuh sebelum disalin agar keputusan akhir (409/303) tak
// terlambat setelah status 200 terkirim.

import (
	"net/http"

	inertia "github.com/romsar/gonertia/v3"
)

// bufferedResponse menahan header/status/body untuk diputuskan belakangan.
type bufferedResponse struct {
	headers http.Header
	code    int
	buffer  []byte
}

func newBufferedResponse() *bufferedResponse {
	return &bufferedResponse{headers: http.Header{}, code: http.StatusOK}
}

func (w *bufferedResponse) Header() http.Header { return w.headers }

func (w *bufferedResponse) WriteHeader(code int) { w.code = code }

func (w *bufferedResponse) Write(p []byte) (int, error) {
	w.buffer = append(w.buffer, p...)
	return len(p), nil
}

func (w *bufferedResponse) copyTo(real http.ResponseWriter) {
	for k, vs := range w.headers {
		for _, v := range vs {
			real.Header().Add(k, v)
		}
	}
	real.WriteHeader(w.code)
	real.Write(w.buffer)
}

// inertiaMiddleware bungkus mux dengan perilaku protokol Inertia.
func (s *Server) inertiaMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !inertia.IsInertiaRequest(r) {
			next.ServeHTTP(w, r)
			return
		}
		w2 := newBufferedResponse()
		next.ServeHTTP(w2, r)

		// pemeriksaan versi: hanya bila klien mengirim versi (bukan fetch awal shell)
		if r.Method == http.MethodGet && r.Header.Get("X-Inertia-Version") != "" &&
			r.Header.Get("X-Inertia-Version") != s.Version {
			w2.headers.Del("X-Inertia")
			w2.headers.Del("Vary")
			w2.headers.Set("X-Inertia-Location", r.URL.RequestURI())
			w2.code = http.StatusConflict
			w2.copyTo(w)
			return
		}

		// respons kosong saat aksi Inertia → kembali ke halaman asal
		if w2.code == http.StatusOK && len(w2.buffer) == 0 {
			referer := r.Header.Get("Referer")
			if referer == "" {
				referer = "/"
			}
			s.I.Redirect(w2, r, referer)
		}

		// 302 setelah PUT/PATCH/DELETE → 303 (semantik protokol)
		if w2.code == http.StatusFound &&
			(r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete) {
			w2.code = http.StatusSeeOther
		}

		w2.copyTo(w)
	})
}
