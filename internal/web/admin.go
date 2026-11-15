package web

// Konsol admin (role=admin, login sama dengan user): statistik, users,
// orders, produk (+upload berkas & stok kunci), mail log, tiket, gateway.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	pay "github.com/jhonoryza/paygateway"
)

// guardAdmin helper: bungkus handler dengan Guard(admin).
func (s *Server) guardAdmin(next func(http.ResponseWriter, *http.Request, auth.User)) http.Handler {
	return auth.Guard(s.DB, s.cookieCfg(), true, next, s.authOnFail("/login", true))
}

// adminDashboard GET /admin — statistik ringkas toko.
// periodeDashboard pilihan rentang waktu kartu order + grafik.
var periodeDashboard = map[string]bool{"": true, "bulanan": true, "mingguan": true, "harian": true}

var bulanSingkat = [12]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

// adminDashboard GET /admin — statistik + grafik. Parameter ?periode=
// (kosong = semua, bulanan = 30 hari, mingguan = 7 hari, harian = hari ini)
// memfilter kartu order serta kedua grafik; kartu pengguna/tiket/mail
// tetap global.
func (s *Server) adminDashboard(w http.ResponseWriter, r *http.Request, u auth.User) {
	periode := r.URL.Query().Get("periode")
	if !periodeDashboard[periode] {
		periode = ""
	}

	now := time.Now().UTC()
	var start time.Time
	var step time.Duration
	var nbucket int
	switch periode {
	case "harian":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		step = time.Hour
		nbucket = 24
	case "mingguan":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -6)
		step = 24 * time.Hour
		nbucket = 7
	case "bulanan":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -29)
		step = 24 * time.Hour
		nbucket = 30
	default:
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -11, 0)
		nbucket = 12
	}

	// batas + label tiap bucket
	type bucket struct {
		start, end time.Time
		label      string
	}
	bounds := make([]bucket, 0, nbucket)
	cur := start
	for i := 0; i < nbucket; i++ {
		var b bucket
		if periode == "" {
			b.start = time.Date(cur.Year(), cur.Month(), 1, 0, 0, 0, 0, time.UTC)
			b.end = b.start.AddDate(0, 1, 0)
			b.label = bulanSingkat[int(b.start.Month())-1]
			cur = b.end
		} else {
			b.start = cur
			b.end = cur.Add(step)
			if periode == "harian" {
				b.label = fmt.Sprintf("%02d", b.start.Hour())
			} else {
				b.label = fmt.Sprintf("%d/%d", b.start.Day(), int(b.start.Month()))
			}
			cur = b.end
		}
		bounds = append(bounds, b)
	}

	// indeks bucket untuk satu waktu
	indeks := func(t time.Time) int {
		if periode == "" {
			i := (t.Year()-bounds[0].start.Year())*12 + int(t.Month()) - int(bounds[0].start.Month())
			if i < 0 {
				i = 0
			}
			if i >= nbucket {
				i = nbucket - 1
			}
			return i
		}
		i := int(t.Sub(bounds[0].start) / step)
		if i < 0 {
			i = 0
		}
		if i >= nbucket {
			i = nbucket - 1
		}
		return i
	}

	series := make([]map[string]any, nbucket)
	for i, b := range bounds {
		series[i] = map[string]any{"label": b.label, "value": 0}
	}
	startStr := start.Format("2006-01-02 15:04:05")
	if periode == "" {
		startStr = "" // grafik "semua" tetap mengikuti 12 bulan terakhir
	}
	if srows, err := s.DB.Query(`SELECT total, created_at FROM orders WHERE status='paid' AND created_at >= ?`, startStr); err == nil {
		for srows.Next() {
			var total int
			var created string
			if srows.Scan(&total, &created) == nil {
				if t, err := time.Parse("2006-01-02 15:04:05", created); err == nil {
					m := series[indeks(t)]
					m["value"] = m["value"].(int) + total
				}
			}
		}
		srows.Close()
	}

	// pie distribusi status order dalam periode
	pie := []map[string]any{}
	pq := `SELECT status, COUNT(*) FROM orders`
	pargs := []any{}
	if periode != "" {
		pq += ` WHERE created_at >= ?`
		pargs = append(pargs, startStr)
	}
	pq += ` GROUP BY status ORDER BY COUNT(*) DESC`
	if prows, err := s.DB.Query(pq, pargs...); err == nil {
		for prows.Next() {
			var st string
			var n int
			if prows.Scan(&st, &n) == nil {
				pie = append(pie, map[string]any{"status": st, "count": n})
			}
		}
		prows.Close()
	}

	// kartu statistik: order mengikuti periode, lainnya global
	statWhere, statArgs := "", []any{}
	if periode != "" {
		statWhere = ` AND created_at >= ?`
		statArgs = append(statArgs, startStr)
	}
	var revenue, ordersPaid, ordersPending, users, ticketsOpen, mailErr int
	var licenseFree, licenseSold int
	s.DB.QueryRow(`SELECT COALESCE(SUM(total),0) FROM orders WHERE status='paid'`+statWhere, statArgs...).Scan(&revenue)
	s.DB.QueryRow(`SELECT COUNT(*) FROM orders WHERE status='paid'`+statWhere, statArgs...).Scan(&ordersPaid)
	s.DB.QueryRow(`SELECT COUNT(*) FROM orders WHERE status='pending'`+statWhere, statArgs...).Scan(&ordersPending)
	s.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role='user'`).Scan(&users)
	s.DB.QueryRow(`SELECT COUNT(*) FROM tickets WHERE status='open'`).Scan(&ticketsOpen)
	s.DB.QueryRow(`SELECT COUNT(*) FROM mail_log WHERE status='error'`).Scan(&mailErr)
	s.DB.QueryRow(`SELECT COUNT(*) FROM license_keys WHERE order_item_id IS NULL`).Scan(&licenseFree)
	s.DB.QueryRow(`SELECT COUNT(*) FROM license_keys WHERE order_item_id IS NOT NULL`).Scan(&licenseSold)

	s.Page(w, r, "Admin/Dashboard", inertia.Props{
		"stats": map[string]any{
			"revenue":        revenue,
			"orders_paid":    ordersPaid,
			"orders_pending": ordersPending,
			"users":          users,
			"tickets_open":   ticketsOpen,
			"mail_error":     mailErr,
			"license_free":   licenseFree,
			"license_sold":   licenseSold,
		},
		"periode": periode,
		"series":  series,
		"pie":     pie,
		"flash":   flashFromQuery(r),
	})
}

// adminUsers GET /admin/users — daftar akun.
// adminUserSorts mapping urutan konsol pengguna.
var adminUserSorts = map[string]string{
	"terbaru": "created_at DESC, id DESC",
	"terlama": "created_at ASC, id ASC",
	"nama":    "name COLLATE NOCASE ASC",
	"email":   "email COLLATE NOCASE ASC",
}

// adminUsers GET /admin/users — daftar pengguna dengan pencarian email/nama,
// filter role/verifikasi, urutan, dan paginasi 12/halaman. Parameter:
// q, role, verified, sort, page.
func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request, u auth.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	role := r.URL.Query().Get("role")
	verified := r.URL.Query().Get("verified")
	sort := r.URL.Query().Get("sort")
	if _, ok := adminUserSorts[sort]; !ok {
		sort = "terbaru"
	}

	var conds []string
	var args []any
	if q != "" {
		esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
		conds = append(conds, `(email LIKE ? ESCAPE '\' OR name LIKE ? ESCAPE '\')`)
		args = append(args, "%"+esc+"%", "%"+esc+"%")
	}
	switch role {
	case "admin", "user":
		conds = append(conds, `role=?`)
		args = append(args, role)
	}
	switch verified {
	case "sudah":
		conds = append(conds, `email_verified_at IS NOT NULL`)
	case "belum":
		conds = append(conds, `email_verified_at IS NULL`)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	const adminPerPage = 12
	var total int
	s.DB.QueryRow(`SELECT COUNT(*) FROM users`+where, args...).Scan(&total)
	lastPage := (total + adminPerPage - 1) / adminPerPage
	if lastPage < 1 {
		lastPage = 1
	}
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p >= 1 && p <= lastPage {
		page = p
	}

	rows, err := s.DB.Query(`SELECT id, email, name, role, COALESCE(email_verified_at,''), created_at FROM users`+where+
		` ORDER BY `+adminUserSorts[sort]+` LIMIT ? OFFSET ?`,
		append(append([]any{}, args...), adminPerPage, (page-1)*adminPerPage)...)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	defer rows.Close()
	var users []map[string]any
	for rows.Next() {
		var id int64
		var email, name, role, verified, created string
		if rows.Scan(&id, &email, &name, &role, &verified, &created) == nil {
			users = append(users, map[string]any{"id": id, "email": email, "name": name, "role": role, "verified": verified != "", "created_at": created})
		}
	}
	s.Page(w, r, "Admin/Users", inertia.Props{
		"users":     users,
		"page":      page,
		"last_page": lastPage,
		"total":     total,
		"filters": map[string]any{
			"q": q, "role": role, "verified": verified, "sort": sort,
		},
		"flash": flashFromQuery(r),
	})
}

// adminOrderSorts mapping urutan konsol order.
var adminOrderSorts = map[string]string{
	"terbaru":  "created_at DESC, id DESC",
	"terlama":  "created_at ASC, id ASC",
	"termahal": "total DESC, id DESC",
	"termurah": "total ASC, id DESC",
}

// adminOrders GET /admin/orders — daftar order dengan pencarian no/email,
// filter status, urutan, dan paginasi 12/halaman. Parameter: q, status, sort, page.
func (s *Server) adminOrders(w http.ResponseWriter, r *http.Request, u auth.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := r.URL.Query().Get("status")
	sort := r.URL.Query().Get("sort")
	if _, ok := adminOrderSorts[sort]; !ok {
		sort = "terbaru"
	}

	var conds []string
	var args []any
	if q != "" {
		esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
		conds = append(conds, `(order_no LIKE ? ESCAPE '\' OR email LIKE ? ESCAPE '\')`)
		args = append(args, "%"+esc+"%", "%"+esc+"%")
	}
	switch status {
	case "paid", "pending":
		conds = append(conds, `status=?`)
		args = append(args, status)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	const adminPerPage = 12
	var total int
	s.DB.QueryRow(`SELECT COUNT(*) FROM orders`+where, args...).Scan(&total)
	lastPage := (total + adminPerPage - 1) / adminPerPage
	if lastPage < 1 {
		lastPage = 1
	}
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p >= 1 && p <= lastPage {
		page = p
	}

	rows, err := s.DB.Query(`SELECT order_no, email, status, subtotal, discount, total, pay_provider, COALESCE(paid_at,''), created_at FROM orders`+where+
		` ORDER BY `+adminOrderSorts[sort]+` LIMIT ? OFFSET ?`,
		append(append([]any{}, args...), adminPerPage, (page-1)*adminPerPage)...)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	defer rows.Close()
	var orders []map[string]any
	for rows.Next() {
		var no, email, status, provider, paid, created string
		var subtotal, discount, total int
		if rows.Scan(&no, &email, &status, &subtotal, &discount, &total, &provider, &paid, &created) == nil {
			orders = append(orders, map[string]any{
				"order_no": no, "email": email, "status": status, "subtotal": subtotal,
				"discount": discount, "total": total, "pay_provider": provider,
				"paid_at": paid, "created_at": created,
			})
		}
	}
	s.Page(w, r, "Admin/Orders", inertia.Props{
		"orders":    orders,
		"page":      page,
		"last_page": lastPage,
		"total":     total,
		"filters":   map[string]any{"q": q, "status": status, "sort": sort},
		"flash":     flashFromQuery(r),
	})
}

// adminOrderPaid POST /admin/orders/{no}/paid — tandai lunas manual (amount=total).
func (s *Server) adminOrderPaid(w http.ResponseWriter, r *http.Request, u auth.User) {
	orderNo := r.PathValue("order_no")
	if s.markOrderPaid(orderNo, r, 0) {
		audit(s.DB, "order.paid", orderNo, "manual oleh "+u.Email)
		s.flashOK(w, r, "/admin/orders", "Order "+orderNo+" ditandai lunas + produk terkirim.")
		return
	}
	s.flashErr(w, r, "/admin/orders", "Gagal - order mungkin sudah lunas/tidak ada.")
}

// adminOrderResend POST /admin/orders/{no}/resend — kirim ulang email produk.
func (s *Server) adminOrderResend(w http.ResponseWriter, r *http.Request, u auth.User) {
	orderNo := r.PathValue("order_no")
	var status string
	if err := s.DB.QueryRow(`SELECT status FROM orders WHERE order_no=?`, orderNo).Scan(&status); err != nil || status != "paid" {
		s.flashErr(w, r, "/admin/orders", "Order harus lunas sebelum produk dikirim ulang.")
		return
	}
	s.onOrderPaid(orderNo, r)
	audit(s.DB, "order.resend", orderNo, "oleh "+u.Email)
	s.flashOK(w, r, "/admin/orders", "Email produk dikirim ulang.")
}

// adminOrderDetail GET /admin/orders/{order_no} — detail order lengkap
// (rincian item, kunci lisensi, tautan unduh ber-token) seperti invoice user.
func (s *Server) adminOrderDetail(w http.ResponseWriter, r *http.Request, u auth.User) {
	o, ok := s.loadOrder(r.PathValue("order_no"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.Page(w, r, "Admin/OrderDetail", inertia.Props{
		"order": o,
		"token": s.orderToken(o.OrderNo),
		"flash": flashFromQuery(r),
	})
}

// ---- produk ----

// productAdmin baris produk untuk konsol.
type productAdmin struct {
	productData
	ExtURL   string `json:"ext_url"`
	FilePath string `json:"file_path"`
	FileName string `json:"file_name"`
	Active   bool   `json:"active"`
	KeyCount int    `json:"key_count"`
}

// adminProductSorts mapping urutan konsol produk (id sebagai pemecah seri).
var adminProductSorts = map[string]string{
	"terbaru":  "p.created_at DESC, p.id DESC",
	"terlama":  "p.created_at ASC, p.id ASC",
	"termurah": "p.price ASC, p.id DESC",
	"termahal": "p.price DESC, p.id DESC",
	"nama":     "p.title COLLATE NOCASE ASC",
}

// adminProducts GET /admin/produk — daftar produk dengan pencarian nama/slug,
// filter kategori/jenis/status, urutan, dan paginasi 12/halaman (klien
// memakai infinite scroll). Parameter: q, kategori, type, status, sort, page.
func (s *Server) adminProducts(w http.ResponseWriter, r *http.Request, u auth.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	kategori := strings.TrimSpace(r.URL.Query().Get("kategori"))
	typ := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	sort := r.URL.Query().Get("sort")
	if _, ok := adminProductSorts[sort]; !ok {
		sort = "terbaru"
	}

	var conds []string
	var args []any
	if q != "" {
		esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
		conds = append(conds, `(p.title LIKE ? ESCAPE '\' OR p.slug LIKE ? ESCAPE '\')`)
		args = append(args, "%"+esc+"%", "%"+esc+"%")
	}
	if kategori != "" {
		conds = append(conds, `LOWER(p.category)=LOWER(?)`)
		args = append(args, kategori)
	}
	switch typ {
	case "file", "key", "link":
		conds = append(conds, `p.type=?`)
		args = append(args, typ)
	}
	switch status {
	case "aktif":
		conds = append(conds, `p.active=1`)
	case "nonaktif":
		conds = append(conds, `p.active=0`)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	const adminPerPage = 12
	var total int
	s.DB.QueryRow(`SELECT COUNT(*) FROM products p`+where, args...).Scan(&total)
	lastPage := (total + adminPerPage - 1) / adminPerPage
	if lastPage < 1 {
		lastPage = 1
	}
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p >= 1 && p <= lastPage {
		page = p
	}

	// opsi filter kategori dari seluruh produk (termasuk nonaktif)
	var categories []string
	if crows, err := s.DB.Query(`SELECT DISTINCT category FROM products WHERE category != '' ORDER BY category`); err == nil {
		for crows.Next() {
			var c string
			if crows.Scan(&c) == nil {
				categories = append(categories, c)
			}
		}
		crows.Close()
	}

	rows, err := s.DB.Query(`SELECT p.id, p.slug, p.title, p.description, p.price, p.type, p.featured, p.active, p.file_path, p.file_name, p.ext_url, p.meta_title, p.meta_description, p.category, p.tags,
		(SELECT COUNT(*) FROM license_keys k WHERE k.product_id=p.id AND k.order_item_id IS NULL)
		FROM products p`+where+` ORDER BY `+adminProductSorts[sort]+` LIMIT ? OFFSET ?`,
		append(append([]any{}, args...), adminPerPage, (page-1)*adminPerPage)...)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	defer rows.Close()
	var products []map[string]any
	for rows.Next() {
		var id int64
		var slug, title, desc, typ, filePath, fileName, extURL, metaTitle, metaDesc, category, tagsJSON string
		var price, featured, active, keyCount int
		if rows.Scan(&id, &slug, &title, &desc, &price, &typ, &featured, &active, &filePath, &fileName, &extURL, &metaTitle, &metaDesc, &category, &tagsJSON, &keyCount) == nil {
			products = append(products, map[string]any{
				"id": id, "slug": slug, "title": title, "description": desc, "price": price,
				"type": typ, "featured": featured == 1, "active": active == 1,
				"file_path": filePath, "file_name": fileName, "ext_url": extURL, "key_count": keyCount,
				"meta_title": metaTitle, "meta_description": metaDesc,
				"category": category, "tags": parseTagsJSON(tagsJSON),
			})
		}
	}
	s.Page(w, r, "Admin/Products", inertia.Props{
		"products":   products,
		"images":     s.allProductImages(r),
		"page":       page,
		"last_page":  lastPage,
		"total":      total,
		"categories": categories,
		"filters": map[string]any{
			"q": q, "kategori": kategori, "type": typ, "status": status, "sort": sort,
		},
		"flash": flashFromQuery(r),
	})
}

// parseTagsForm ubah input tag dipisah koma/baris baru menjadi JSON array
// bersih: trim, kosong dibuang, duplikat dilebur (case-insensitive), maks 10
// tag dan 40 karakter per tag.
func parseTagsForm(raw string) string {
	var tags []string
	seen := map[string]bool{}
	for _, t := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' }) {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if r := []rune(t); len(r) > 40 {
			t = string(r[:40])
		}
		if key := strings.ToLower(t); seen[key] {
			continue
		} else {
			seen[key] = true
		}
		tags = append(tags, t)
	}
	if tags == nil {
		tags = []string{}
	}
	b, _ := json.Marshal(tags)
	return string(b)
}

// adminProductSave POST /admin/produk (create) dan /admin/produk/{id} (update).
// Upload multipart: field "file" (opsional, hanya type=file).
func (s *Server) adminProductSave(w http.ResponseWriter, r *http.Request, u auth.User) {
	var id int64
	if raw := r.PathValue("id"); raw != "" {
		var err error
		id, err = pathInt(raw)
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}

	slug := strings.ToLower(strings.TrimSpace(r.PostFormValue("slug")))
	title := strings.TrimSpace(r.PostFormValue("title"))
	description := strings.TrimSpace(r.PostFormValue("description"))
	price, _ := pathInt(r.PostFormValue("price"))
	ptype := r.PostFormValue("type")
	featured := r.PostFormValue("featured") == "1"
	active := r.PostFormValue("active") != "0"
	extURL := strings.TrimSpace(r.PostFormValue("ext_url"))
	metaTitle := strings.TrimSpace(r.PostFormValue("meta_title"))
	metaDesc := strings.TrimSpace(r.PostFormValue("meta_description"))
	category := strings.TrimSpace(r.PostFormValue("kategori"))
	tagsJSON := parseTagsForm(r.PostFormValue("tags"))

	errs := inertia.ValidationErrors{}
	if slug == "" || !validEmail(slug+"@x.y") || strings.ContainsAny(slug, " ") {
		errs["slug"] = "slug wajib, huruf kecil/angka/strip"
	}
	if title == "" {
		errs["title"] = "judul wajib diisi"
	}
	if price <= 0 {
		errs["price"] = "harga harus > 0"
	}
	if ptype != "file" && ptype != "key" && ptype != "link" {
		ptype = "file"
	}
	if ptype == "link" && extURL == "" {
		errs["ext_url"] = "tautan wajib untuk produk link"
	}

	// unggahan berkas (opsional pada update, wajib di create type=file)
	var savedPath, savedName string
	file, header, ferr := r.FormFile("file")
	if ferr == nil {
		defer file.Close()
		ext := filepath.Ext(header.Filename)
		name := hex.EncodeToString(randomBytes(8)) + ext
		dst := filepath.Join(s.filesDir(), name)
		if err := os.MkdirAll(s.filesDir(), 0o755); err != nil {
			jsonError(w, http.StatusInternalServerError, "kesalahan server")
			return
		}
		out, err := os.Create(dst)
		if err == nil {
			if _, err := io.Copy(out, file); err == nil {
				out.Close()
				savedPath, savedName = name, filepath.Base(header.Filename)
			} else {
				out.Close()
				os.Remove(dst)
			}
		}
	}
	if ptype == "file" && id == 0 && savedPath == "" {
		errs["file"] = "berkas wajib diunggah untuk produk file"
	}

	if len(errs) > 0 {
		s.renderWithErrors(w, r, "Admin/Products", nil, errs)
		return
	}

	if id == 0 {
		_, err := s.DB.Exec(`INSERT INTO products (slug, title, description, price, type, featured, active, file_path, file_name, ext_url, meta_title, meta_description, category, tags)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			slug, title, description, price, ptype, boolInt(featured), boolInt(active), savedPath, savedName, extURL, metaTitle, metaDesc, category, tagsJSON)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				s.renderWithErrors(w, r, "Admin/Products", nil, inertia.ValidationErrors{"slug": "slug sudah dipakai"})
				return
			}
			jsonError(w, http.StatusInternalServerError, "kesalahan server")
			return
		}
		audit(s.DB, "produk.buat", slug, fmt.Sprintf("%s %d", ptype, price))
		s.flashOK(w, r, "/admin/produk", "Produk dibuat.")
		return
	}

	// update: berkas baru menimpa path bila ada
	if savedPath != "" {
		var old string
		s.DB.QueryRow(`SELECT file_path FROM products WHERE id=?`, id).Scan(&old)
		if old != "" && old != savedPath {
			os.Remove(filepath.Join(s.filesDir(), old))
		}
		s.DB.Exec(`UPDATE products SET slug=?, title=?, description=?, price=?, type=?, featured=?, active=?, file_path=?, file_name=?, ext_url=?, category=?, tags=?, updated_at=datetime('now') WHERE id=?`,
			slug, title, description, price, ptype, boolInt(featured), boolInt(active), savedPath, savedName, extURL, category, tagsJSON, id)
	} else {
		s.DB.Exec(`UPDATE products SET slug=?, title=?, description=?, price=?, type=?, featured=?, active=?, ext_url=?, category=?, tags=?, updated_at=datetime('now') WHERE id=?`,
			slug, title, description, price, ptype, boolInt(featured), boolInt(active), extURL, category, tagsJSON, id)
	}
	audit(s.DB, "produk.update", slug, fmt.Sprintf("id %d", id))
	s.flashOK(w, r, "/admin/produk", "Produk diperbarui.")
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// adminProductKeys POST /admin/produk/{id}/keys — tambah stok kunci massal.
func (s *Server) adminProductKeys(w http.ResponseWriter, r *http.Request, u auth.User) {
	id, err := pathInt(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var slug string
	if err := s.DB.QueryRow(`SELECT slug FROM products WHERE id=?`, id).Scan(&slug); err != nil {
		http.NotFound(w, r)
		return
	}
	keys := strings.Split(strings.ReplaceAll(r.PostFormValue("keys"), "\r\n", "\n"), "\n")
	added := 0
	for _, k := range keys {
		if k = strings.TrimSpace(k); k != "" {
			s.DB.Exec(`INSERT INTO license_keys (product_id, key) VALUES (?,?)`, id, k)
			added++
		}
	}
	audit(s.DB, "produk.kunci", slug, fmt.Sprintf("+%d", added))
	s.flashOK(w, r, "/admin/produk", fmt.Sprintf("%d kunci ditambahkan.", added))
}

// adminProductToggle POST /admin/produk/{id}/toggle — aktif/nonaktif.
func (s *Server) adminProductToggle(w http.ResponseWriter, r *http.Request, u auth.User) {
	id, err := pathInt(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.DB.Exec(`UPDATE products SET active = 1 - active, updated_at=datetime('now') WHERE id=?`, id)
	audit(s.DB, "produk.toggle", fmtInt(id), "")
	s.flashOK(w, r, "/admin/produk", "Status produk diubah.")
}

// ---- mail log & tiket admin ----

func (s *Server) adminMailLog(w http.ResponseWriter, r *http.Request, u auth.User) {
	rows, err := s.DB.Query(`SELECT to_email, subject, status, error, created_at FROM mail_log ORDER BY id DESC LIMIT 200`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	defer rows.Close()
	var logs []map[string]any
	for rows.Next() {
		var to, subject, status, errMsg, created string
		if rows.Scan(&to, &subject, &status, &errMsg, &created) == nil {
			logs = append(logs, map[string]any{
				"to": to, "subject": subject, "status": status, "error": errMsg, "created_at": created,
			})
		}
	}
	s.Page(w, r, "Admin/MailLog", inertia.Props{"logs": logs, "flash": flashFromQuery(r)})
}

// adminTickets GET /admin/tiket — seluruh tiket.
// adminTicketSorts mapping urutan konsol tiket.
var adminTicketSorts = map[string]string{
	"terbaru": "t.updated_at DESC, t.id DESC",
	"terlama": "t.updated_at ASC, t.id ASC",
	"subjek":  "t.subject COLLATE NOCASE ASC",
}

// adminTickets GET /admin/tiket — daftar tiket dengan pencarian subjek/email/
// nomor order, filter status, urutan, dan paginasi 12/halaman. Parameter:
// q, status, sort, page.
func (s *Server) adminTickets(w http.ResponseWriter, r *http.Request, u auth.User) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := r.URL.Query().Get("status")
	sort := r.URL.Query().Get("sort")
	if _, ok := adminTicketSorts[sort]; !ok {
		sort = "terbaru"
	}

	var conds []string
	var args []any
	if q != "" {
		esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
		conds = append(conds, `(t.subject LIKE ? ESCAPE '\' OR u.email LIKE ? ESCAPE '\' OR t.order_no LIKE ? ESCAPE '\')`)
		args = append(args, "%"+esc+"%", "%"+esc+"%", "%"+esc+"%")
	}
	switch status {
	case "open", "answered", "closed":
		conds = append(conds, `t.status=?`)
		args = append(args, status)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	const adminPerPage = 12
	var total int
	s.DB.QueryRow(`SELECT COUNT(*) FROM tickets t JOIN users u ON u.id = t.user_id`+where, args...).Scan(&total)
	lastPage := (total + adminPerPage - 1) / adminPerPage
	if lastPage < 1 {
		lastPage = 1
	}
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p >= 1 && p <= lastPage {
		page = p
	}

	rows, err := s.DB.Query(`SELECT t.id, t.order_no, t.subject, t.status, t.updated_at, u.email
		FROM tickets t JOIN users u ON u.id = t.user_id`+where+` ORDER BY `+adminTicketSorts[sort]+` LIMIT ? OFFSET ?`,
		append(append([]any{}, args...), adminPerPage, (page-1)*adminPerPage)...)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	defer rows.Close()
	var tickets []map[string]any
	for rows.Next() {
		var id int64
		var orderNo, subject, status, updated, email string
		if rows.Scan(&id, &orderNo, &subject, &status, &updated, &email) == nil {
			tickets = append(tickets, map[string]any{
				"id": id, "order_no": orderNo, "subject": subject, "status": status,
				"updated_at": updated, "email": email,
			})
		}
	}
	s.Page(w, r, "Admin/Tickets", inertia.Props{
		"tickets":   tickets,
		"page":      page,
		"last_page": lastPage,
		"total":     total,
		"filters":   map[string]any{"q": q, "status": status, "sort": sort},
		"flash":     flashFromQuery(r),
	})
}

// adminTicketDetail GET /admin/tiket/{id}.
func (s *Server) adminTicketDetailPage(w http.ResponseWriter, r *http.Request, u auth.User) {
	id, err := pathInt(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var t ticketData
	var owner int64
	var ownerEmail string
	err = s.DB.QueryRow(`SELECT t.id, t.user_id, t.order_no, t.subject, t.status, t.created_at, t.updated_at, u.email
		FROM tickets t JOIN users u ON u.id=t.user_id WHERE t.id=?`, id).
		Scan(&t.ID, &owner, &t.OrderNo, &t.Subject, &t.Status, &t.CreatedAt, &t.UpdatedAt, &ownerEmail)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	msgs, _ := s.DB.Query(`SELECT id, sender, body, created_at FROM ticket_messages WHERE ticket_id=? ORDER BY id`, id)
	if msgs != nil {
		defer msgs.Close()
		for msgs.Next() {
			var m ticketMessage
			if msgs.Scan(&m.ID, &m.Sender, &m.Body, &m.CreatedAt) == nil {
				t.Messages = append(t.Messages, m)
			}
		}
	}
	s.Page(w, r, "Admin/TicketDetail", inertia.Props{"ticket": t, "email": ownerEmail, "flash": flashFromQuery(r)})
}

// adminTicketReply POST /admin/tiket/{id} — balas sbg admin → status answered.
func (s *Server) adminTicketReply(w http.ResponseWriter, r *http.Request, u auth.User) {
	id, err := pathInt(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	body := strings.TrimSpace(r.PostFormValue("body"))
	if body == "" {
		s.renderWithErrors(w, r, "Admin/TicketDetail", nil, inertia.ValidationErrors{"body": "pesan wajib diisi"})
		return
	}
	s.DB.Exec(`INSERT INTO ticket_messages (ticket_id, sender, body) VALUES (?,?,?)`, id, "admin", body)
	s.DB.Exec(`UPDATE tickets SET status='answered', updated_at=datetime('now') WHERE id=?`, id)
	audit(s.DB, "tiket.balas", fmtInt(id), "oleh "+u.Email)
	s.Redirect(w, r, "/admin/tiket/"+fmtInt(id))
}

// adminTicketClose POST /admin/tiket/{id}/close.
func (s *Server) adminTicketClose(w http.ResponseWriter, r *http.Request, u auth.User) {
	id, err := pathInt(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.DB.Exec(`UPDATE tickets SET status='closed', updated_at=datetime('now') WHERE id=?`, id)
	audit(s.DB, "tiket.tutup", fmtInt(id), "")
	s.flashOK(w, r, "/admin/tiket", "Tiket ditutup.")
}

// ---- gateway (paygateway) ----

// adminGateways GET /admin/gateway — daftar provider + cfg (secret masked).
func (s *Server) adminGateways(w http.ResponseWriter, r *http.Request, u auth.User) {
	active := getSetting(s.DB, "payment_provider")
	var providers []map[string]any
	for _, p := range pay.List() {
		cfg := providerCfg(s.DB, p.Name())
		fields := []map[string]any{}
		for _, f := range p.Fields() {
			val := cfg[f.Key]
			if f.Secret && val != "" {
				val = "•••"
			}
			fields = append(fields, map[string]any{"key": f.Key, "label": f.Label, "secret": f.Secret, "value": val})
		}
		providers = append(providers, map[string]any{
			"name": p.Name(), "label": p.Label(), "fields": fields,
			"simulated": pay.IsSimulated(p), "configured": len(cfg) > 0,
		})
	}
	s.Page(w, r, "Admin/Gateways", inertia.Props{
		"active":    active,
		"providers": providers,
		"flash":     flashFromQuery(r),
	})
}

// adminGatewayConfig POST /admin/gateway/config — simpan cfg (empty/••• = tetap).
func (s *Server) adminGatewayConfig(w http.ResponseWriter, r *http.Request, u auth.User) {
	name := r.PostFormValue("provider")
	p, ok := pay.Get(name)
	if !ok {
		s.flashErr(w, r, "/admin/gateway", "Provider tidak dikenal.")
		return
	}
	cfg := providerCfg(s.DB, name)
	for _, f := range p.Fields() {
		v := r.PostFormValue(f.Key)
		if v == "" || v == "•••" {
			continue
		}
		cfg[f.Key] = v
	}
	b, _ := json.Marshal(cfg)
	setSetting(s.DB, "payment_cfg."+name, string(b))
	audit(s.DB, "gateway.config", name, "oleh "+u.Email)
	s.flashOK(w, r, "/admin/gateway", "Konfigurasi "+p.Label()+" disimpan.")
}

// adminGatewayActivate POST /admin/gateway/activate.
func (s *Server) adminGatewayActivate(w http.ResponseWriter, r *http.Request, u auth.User) {
	name := r.PostFormValue("provider")
	if _, ok := pay.Get(name); !ok {
		s.flashErr(w, r, "/admin/gateway", "Provider tidak dikenal.")
		return
	}
	setSetting(s.DB, "payment_provider", name)
	audit(s.DB, "gateway.activate", name, "oleh "+u.Email)
	s.flashOK(w, r, "/admin/gateway", "Gateway aktif: "+name)
}
