// store — server toko digital single-tenant.
//
//	store serve      jalankan web server
//	store migrate    jalankan migrasi database lalu keluar
//	store seed       buat akun admin + produk contoh (dev)
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
			srv.Maintain()
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
	pass := os.Getenv("STORE_SEED_ADMIN_PASSWORD")
	if pass == "" {
		pass = "admin12345"
	}
	if exists == 0 {
		hash, err := auth.HashPassword(pass)
		must(err)
		r, err := d.Exec(`INSERT INTO users (email, name, password, role, email_verified_at) VALUES (?,?,?, 'admin', datetime('now'))`, email, "Admin", hash)
		must(err)
		id, _ := r.LastInsertId()
		fmt.Printf("admin dibuat: %s / %s (id %d) - GANTI password segera\n", email, pass, id)
	} else {
		fmt.Println("admin sudah ada - pembuatan admin dilewati")
	}

	// produk contoh + dummy hingga 50 item (idempoten: skip bila slug sudah ada)
	samples := []struct {
		slug, title, desc, category, tags string
		price                             int
		typ                               string
	}{
		{"paket-template-landing", "Paket Template Landing Page", "5 template landing page siap pakai (HTML + Tailwind), termasuk dokumentasi.", "Template", `["unduhan","siap-pakai","favorit"]`, 149000, "file"},
		{"e-book-go-golang", "E-Book: Belajar Go dari Nol", "120 halaman PDF bahasa Indonesia: sintaks dasar sampai deploy ke VPS.", "E-Book", `["unduhan","belajar","promo"]`, 79000, "file"},
		{"kunci-tools-premium", "Kunci Aktivasi Tools Premium", "Kunci lisensi perangkat lunak desktop, 1 order = 1 kunci.", "Lisensi", `["lisensi","aktivasi","premium"]`, 250000, "key"},
		{"notion-dashboard-umkm", "Notion Dashboard UMKM", "Template Notion manajemen usaha kecil - tautan duplikat ke workspace Anda.", "Aplikasi", `["akses","favorit"]`, 59000, "link"},
	}
	for _, s := range samples {
		var n int
		d.QueryRow(`SELECT COUNT(*) FROM products WHERE slug=?`, s.slug).Scan(&n)
		if n > 0 {
			continue
		}
		if _, err := d.Exec(`INSERT INTO products (slug, title, description, price, type, featured, category, tags) VALUES (?,?,?,?,?,1,?,?)`,
			s.slug, s.title, s.desc, s.price, s.typ, s.category, s.tags); err != nil {
			must(err)
		}
	}

	// dummy katalog hingga total 50 produk: nama, jenis, dan harga bervariasi
	// agar katalog dev terasa nyata. Deterministik (tanpa random) agar mudah
	// diuji; produk file diberi berkas contoh, produk key diberi stok kunci.
	presets := []struct{ nama, niche string }{
		{"Preset Lightroom Moody", "fotografi"}, {"UI Kit Dashboard Figma", "desain antarmuka"},
		{"Font Bundle Kufi Modern", "tipografi arab"}, {"E-Book Copywriting Persuasif", "menjual lewat tulisan"},
		{"Template Canva UMKM", "konten media sosial"}, {"Plugin WooCommerce Merdeka", "toko online WordPress"},
		{"Preset Capcut Cinematic", "video pendek"}, {"Mockup Kaos Polos", "presentasi desain"},
		{"Icon Pack Line Art", "ikon antarmuka"}, {"Template CV Kreatif", "lamaran kerja"},
		{"Spreadsheet Keuangan UMKM", "pembukuan sederhana"}, {"Koleksi Wallpaper 4K", "desktop dan ponsel"},
		{"Preset Photoshop Retouch", "editan potret"}, {"Template Landing Page SaaS", "peluncuran produk"},
		{"E-Book SEO Lokal", "peringkat Google Maps"}, {"Koleksi Font Script", "undangan dan branding"},
		{"UI Kit Mobile Banking", "aplikasi finansial"}, {"Preset Instagram Feed", "estetika unggahan"},
		{"Template Notion Habit Tracker", "kebiasaan harian"}, {"Mockup Kemasan Produk", "fotografi produk"},
		{"E-Book Freelance dari Nol", "karier lepas"}, {"Preset After Effects Transition", "montase video"},
		{"Icon Pack Doodle", "ilustrasi lucu"}, {"Template Website Sekolah", "profil lembaga"},
		{"Koleksi Font Display", "judul poster"}, {"Spreadsheet Stok Barang", "inventaris toko"},
		{"Preset Lightroom Wedding", "foto pernikahan"}, {"UI Kit E-Commerce", "katalog belanja"},
		{"Template Pitch Deck Investor", "presentasi bisnis"}, {"E-Book Membangun Personal Brand", "media sosial"},
		{"Plugin Form Builder Ringan", "formulir kontak"}, {"Preset Capcut Fast Edit", "video reels"},
		{"Mockup Perangkat 3D", "mockup aplikasi"}, {"Koleksi Ilustrasi Flat", "ilustrasi web"},
		{"Template Proposal Klien", "dokumen penawaran"}, {"E-Book Ads Marketplace", "iklan daring"},
		{"Preset Photoshop Cinematic", "warna film"}, {"UI Kit Travel App", "aplikasi perjalanan"},
		{"Template Katalog Produk", "brosur digital"}, {"Koleksi Sound Effect Vlog", "audio video"},
		{"Font Display Retro", "nuansa nostalgia"}, {"Spreadsheet Absensi Karyawan", "rekap kehadiran"},
		{"Preset Lightroom Food", "foto kuliner"}, {"Template Undangan Digital", "acara pernikahan"},
		{"E-Book Jualan di TikTok", "konten penjualan"}, {"Mockup Sticker Pack", "stiker chat"},
	}
	base := 39000
	target := 50
	created := 0
	for i, pr := range presets {
		var total int
		d.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&total)
		if total >= target {
			break
		}
		typ := []string{"file", "file", "key", "link"}[i%4]
		price := base + (i%9)*25000 + (i%3)*5000
		feat := 0
		if i < 6 {
			feat = 1 // sebagian dummy tampil di beranda
		}
		if typ == "file" {
			price += 20000 // produk berkas sedikit lebih mahal
		}
		slug := strings.ReplaceAll(strings.ToLower(pr.nama), " ", "-")
		var n int
		d.QueryRow(`SELECT COUNT(*) FROM products WHERE slug=?`, slug).Scan(&n)
		if n > 0 {
			continue
		}
		desc := map[string]string{
			"file": pr.nama + " untuk kebutuhan " + pr.niche + ". Paket lengkap berkas siap pakai, mudah disesuaikan, dan termasuk panduan pemakaian.",
			"key":  "Kunci lisensi " + pr.niche + " - satu order berisi satu kunci aktivasi resmi yang dapat dipakai pada satu perangkat.",
			"link": "Akses " + pr.niche + " berupa tautan duplikat ke workspace Anda. Buka kapan saja setelah pembayaran terverifikasi.",
		}[typ]
		category := "Lainnya"
		for _, c := range []string{"Preset", "UI Kit", "Font", "E-Book", "Template", "Mockup", "Icon", "Spreadsheet", "Plugin", "Koleksi"} {
			if strings.HasPrefix(pr.nama, c) {
				category = c
				break
			}
		}
		baseTags := map[string][]string{
			"file": {"unduhan", "siap-pakai"},
			"key":  {"lisensi", "aktivasi"},
			"link": {"akses", "tautan"},
		}
		tags := append(baseTags[typ], []string{"favorit", "promo", "premium", "baru"}[i%4])
		tagsJSON, err := json.Marshal(tags)
		if err != nil {
			tagsJSON = []byte("[]")
		}
		if _, err := d.Exec(`INSERT INTO products (slug, title, description, price, type, featured, category, tags) VALUES (?,?,?,?,?,?,?,?)`,
			slug, pr.nama, desc, price, typ, feat, category, string(tagsJSON)); err != nil {
			must(err)
		}
		// pelengkap sesuai jenis: berkas contoh utk file, stok kunci utk key,
		// tautan tujuan utk link - agar alur dev benar-benar bisa dicoba.
		var prodID int64
		d.QueryRow(`SELECT id FROM products WHERE slug=?`, slug).Scan(&prodID)
		switch typ {
		case "file":
			fname := slug + ".txt"
			if err := os.MkdirAll(filepath.Join(cfg.Dir, "files"), 0o755); err == nil {
				if err := os.WriteFile(filepath.Join(cfg.Dir, "files", fname),
					[]byte("Berkas dummy untuk "+pr.nama+" - ganti dengan berkas asli di dashboard admin."), 0o644); err == nil {
					d.Exec(`UPDATE products SET file_path=?, file_name=? WHERE id=?`, fname, pr.nama+".txt", prodID)
				}
			}
		case "key":
			for k := 0; k < 10; k++ {
				d.Exec(`INSERT INTO license_keys (product_id, key) VALUES (?,?)`,
					prodID, fmt.Sprintf("%s-%04d", strings.ToUpper(slug[:12]), 1000+k))
			}
		case "link":
			d.Exec(`UPDATE products SET ext_url=? WHERE id=?`, "https://contoh.id/"+slug, prodID)
		}
		created++
		_ = base
	}
	fmt.Printf("produk contoh siap (total %d di katalog, %d dummy baru)\n", target, created)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
