// store — server toko digital single-tenant.
//
//	store serve      jalankan web server
//	store migrate    jalankan migrasi database lalu keluar
//	store seed       buat akun admin + produk contoh (dev)
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	"github.com/jhonoryza/go-self-digital-store/internal/config"
	"github.com/jhonoryza/go-self-digital-store/internal/db"
	"github.com/jhonoryza/go-self-digital-store/internal/mail"
	"github.com/jhonoryza/go-self-digital-store/internal/web"
)

func main() {
	cfg := config.Load()
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "serve":
		serve(cfg)
	case "migrate":
		d, err := db.Open(cfg.Dir)
		must(err)
		must(db.Migrate(d))
		fmt.Println("migrasi selesai")
	case "seed":
		seed(cfg)
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "pakai: store serve|migrate|seed")
	os.Exit(2)
}

func serve(cfg config.Config) {
	d, err := db.Open(cfg.Dir)
	must(err)
	must(db.Migrate(d))

	ml := mail.New(d, mail.Config(cfg.SMTP))
	i, version := web.NewInertia(cfg)
	srv := web.New(d, cfg, ml, i, version)

	// pekerja latar: sweep rate limiter + backup berkala
	go func() {
		t := time.NewTicker(15 * time.Minute)
		defer t.Stop()
		for range t.C {
			web.Sweep()
			if cfg.SMTP.Enabled() {
				if err := db.Backup(d, cfg.Dir, backupKeep()); err != nil {
					log.Printf("backup: %v", err)
				}
			}
		}
	}()

	log.Printf("%s berjalan di %s (dir %s)", cfg.AppName, cfg.Addr, cfg.Dir)
	log.Fatal(http.ListenAndServe(cfg.Addr, srv.Mux))
}

func backupKeep() int {
	if v := os.Getenv("STORE_BACKUP_KEEP"); v != "" {
		n := 0
		fmt.Sscanf(v, "%d", &n)
		if n > 0 {
			return n
		}
	}
	return 7
}

// seed buat admin awal + produk contoh; idempoten (skip bila email sudah ada).
func seed(cfg config.Config) {
	d, err := db.Open(cfg.Dir)
	must(err)
	must(db.Migrate(d))

	email := os.Getenv("STORE_SEED_ADMIN_EMAIL")
	if email == "" {
		email = "admin@example.com"
	}
	var exists int
	d.QueryRow(`SELECT COUNT(*) FROM users WHERE email=?`, email).Scan(&exists)
	if exists > 0 {
		fmt.Println("admin sudah ada, seed dilewati")
		return
	}
	pass := os.Getenv("STORE_SEED_ADMIN_PASSWORD")
	if pass == "" {
		pass = "admin12345"
	}
	hash, err := auth.HashPassword(pass)
	must(err)
	r, err := d.Exec(`INSERT INTO users (email, name, password, role) VALUES (?,?,?, 'admin')`, email, "Admin", hash)
	must(err)
	id, _ := r.LastInsertId()
	fmt.Printf("admin dibuat: %s / %s (id %d) — GANTI password segera\n", email, pass, id)

	// produk contoh (idempoten: skip bila slug sudah ada)
	samples := []struct {
		slug, title, desc string
		price             int
		typ               string
	}{
		{"paket-template-landing", "Paket Template Landing Page", "5 template landing page siap pakai (HTML + Tailwind), termasuk dokumentasi.", 149000, "file"},
		{"e-book-go-golang", "E-Book: Belajar Go dari Nol", "120 halaman PDF bahasa Indonesia: sintaks dasar sampai deploy ke VPS.", 79000, "file"},
		{"kunci-tools-premium", "Kunci Aktivasi Tools Premium", "Kunci lisensi perangkat lunak desktop, 1 order = 1 kunci.", 250000, "key"},
		{"notion-dashboard-umkm", "Notion Dashboard UMKM", "Template Notion manajemen usaha kecil — tautan duplikat ke workspace Anda.", 59000, "link"},
	}
	for _, s := range samples {
		var n int
		d.QueryRow(`SELECT COUNT(*) FROM products WHERE slug=?`, s.slug).Scan(&n)
		if n > 0 {
			continue
		}
		_, err := d.Exec(`INSERT INTO products (slug, title, description, price, type, featured) VALUES (?,?,?,?,?,1)`,
			s.slug, s.title, s.desc, s.price, s.typ)
		must(err)
	}
	fmt.Println("produk contoh siap (4 item)")
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
