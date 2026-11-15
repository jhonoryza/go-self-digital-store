package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jhonoryza/go-self-digital-store/internal/config"
	"github.com/jhonoryza/go-self-digital-store/internal/db"
	"github.com/jhonoryza/go-self-digital-store/internal/mail"
)

func TestDebugLoginToken(t *testing.T) {
	dir := t.TempDir()
	d, err := db.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(d); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Dir: dir, AppName: "DBG"}
	i, version := NewInertia(cfg)
	s := New(d, cfg, mail.New(d, mail.Config{}), i, version)
	ts := httptest.NewServer(s.Mux)
	defer ts.Close()

	email := "dbg@uji.id"
	d.Exec(`INSERT INTO users (email, name, password, role, email_verified_at) VALUES (?,?,?, 'user', datetime('now'))`,
		email, "D", "x")
	linkToken, code, err := s.createLoginTokens(email)
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	var storedHash, expires string
	if err := d.QueryRow(`SELECT id, token_hash, expires_at FROM login_tokens WHERE email=? ORDER BY id DESC LIMIT 1`, email).
		Scan(&id, &storedHash, &expires); err != nil {
		t.Fatal(err)
	}
	t.Logf("linkToken len=%d head=%s..., storedHash head=%s..., id=%d, expires=%s", len(linkToken), linkToken[:10], storedHash[:10], id, expires)
	t.Logf("hashOTP(linkToken) head=%s...", hashOTP(linkToken)[:10])
	t.Logf("code=%s", code)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/login/link/"+linkToken, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	var attempts int
	var used bool
	d.QueryRow(`SELECT attempts, used_at IS NOT NULL FROM login_tokens WHERE id=?`, id).Scan(&attempts, &used)
	t.Logf("consume: status=%d loc=%q attempts=%d used=%v", res.StatusCode, res.Header.Get("Location"), attempts, used)
}
