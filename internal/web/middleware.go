package web

// Middleware: security headers, CORS (split-origin CF Pages), rate limit
// in-memory per-IP, resolusi IP klien (proxy-aware).

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// secHeaders header keamanan dasar untuk semua respons.
func (s *Server) secHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next.ServeHTTP(w, r)
	})
}

// cors CORS untuk origin frontend (CF Pages) bila dikonfigurasi.
// Inertia klien membaca header X-Inertia* — wajib di-expose.
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(s.Cfg.CorsOrigins) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		allowed := false
		for _, o := range s.Cfg.CorsOrigins {
			if origin == o {
				allowed = true
				break
			}
		}
		if allowed {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "X-Inertia, X-Inertia-Version, X-Inertia-Partial-Data, X-Inertia-Partial-Except, X-Inertia-Partial-Component, X-Requested-With, X-CSRF-Token, Content-Type, Accept")
			h.Set("Access-Control-Expose-Headers", "X-Inertia, X-Inertia-Location, X-Inertia-Version, X-CSRF-Token, Vary")
			h.Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---- rate limit in-memory per-IP per-endpoint ----

type ipLimiter struct {
	mu    sync.Mutex
	calls map[string][]time.Time
}

var limiter = &ipLimiter{calls: map[string][]time.Time{}}

// quota permintaan per menit per endpoint.
var ipLimits = map[string]int{
	"callback": 120,
	"login":    10,
	"register": 5,
	"reset":    5,
	"checkout": 30,
	"download": 30,
	"magic":    20,
	"global":   600,
}

// ipLimited true bila melewati kuota (dipanggil sebelum handler).
func ipLimited(kind, ip string) bool {
	quota, ok := ipLimits[kind]
	if !ok {
		quota = 60
	}
	now := time.Now()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	key := kind + "|" + ip
	recent := limiter.calls[key][:0]
	for _, t := range limiter.calls[key] {
		if now.Sub(t) < time.Minute {
			recent = append(recent, t)
		}
	}
	limiter.calls[key] = recent
	if len(recent) >= quota {
		return true
	}
	limiter.calls[key] = append(limiter.calls[key], now)
	return false
}

// limiterSweep buang entri kedaluwarsa (dipanggil worker latar).
func limiterSweep() {
	now := time.Now()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	for k, ts := range limiter.calls {
		recent := ts[:0]
		for _, t := range ts {
			if now.Sub(t) < time.Minute {
				recent = append(recent, t)
			}
		}
		if len(recent) == 0 {
			delete(limiter.calls, k)
		} else {
			limiter.calls[k] = recent
		}
	}
}

// clientIP ambil IP klien; XFF paling kanan hanya dipercaya bila di belakang
// proxy sendiri (STORE_TRUSTED_PROXY) — klien tak bisa memalsukannya.
func (s *Server) clientIP(r *http.Request) string {
	if s.Cfg.TrustedProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// limited wrapper rate-limit per endpoint.
func (s *Server) limited(kind string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ipLimited(kind, s.clientIP(r)) {
			http.Error(w, "terlalu banyak permintaan, coba lagi nanti", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// recover panic per-request -> 500, agar satu bug tak menjatuhkan server.
func recover500(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logErr("panic: %v", err)
				http.Error(w, "kesalahan server", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Sweep buang entri rate limiter kedaluwarsa (dipanggil worker latar).
func Sweep() { limiterSweep() }
