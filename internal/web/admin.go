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
	"strings"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	pay "github.com/jhonoryza/paygateway"
)

// guardAdmin helper: bungkus handler dengan Guard(admin).
func (s *Server) guardAdmin(next func(http.ResponseWriter, *http.Request, auth.User)) http.Handler {
	return auth.Guard(s.DB, s.cookieCfg(), true, next, s.authOnFail("/login", true))
}

// adminDashboard GET /admin — statistik ringkas toko.
func (s *Server) adminDashboard(w http.ResponseWriter, r *http.Request, u auth.User) {
	var revenue, ordersPaid, ordersPending, users, ticketsOpen, mailErr int
	var licenseFree, licenseSold int
	s.DB.QueryRow(`SELECT COALESCE(SUM(total),0) FROM orders WHERE status='paid'`).Scan(&revenue)
	s.DB.QueryRow(`SELECT COUNT(*) FROM orders WHERE status='paid'`).Scan(&ordersPaid)
	s.DB.QueryRow(`SELECT COUNT(*) FROM orders WHERE status='pending'`).Scan(&ordersPending)
	s.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role='user'`).Scan(&users)
	s.DB.QueryRow(`SELECT COUNT(*) FROM tickets WHERE status='open'`).Scan(&ticketsOpen)
	s.DB.QueryRow(`SELECT COUNT(*) FROM mail_log WHERE status='error'`).Scan(&mailErr)
	s.DB.QueryRow(`SELECT COUNT(*) FROM license_keys WHERE order_item_id IS NULL`).Scan(&licenseFree)
	s.DB.QueryRow(`SELECT COUNT(*) FROM license_keys WHERE order_item_id IS NOT NULL`).Scan(&licenseSold)

	// order terbaru
	rows, err := s.DB.Query(`SELECT order_no, email, status, total, created_at FROM orders ORDER BY id DESC LIMIT 8`)
	var recent []map[string]any
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var no, email, status, created string
			var total int
			if rows.Scan(&no, &email, &status, &total, &created) == nil {
				recent = append(recent, map[string]any{
					"order_no": no, "email": email, "status": status, "total": total, "created_at": created,
				})
			}
		}
	}

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
		"recent": recent,
		"flash":  flashFromQuery(r),
	})
}

// adminUsers GET /admin/users — daftar akun.
func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request, u auth.User) {
	rows, err := s.DB.Query(`SELECT id, email, name, role, created_at FROM users ORDER BY id DESC LIMIT 200`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	defer rows.Close()
	var users []map[string]any
	for rows.Next() {
		var id int64
		var email, name, role, created string
		if rows.Scan(&id, &email, &name, &role, &created) == nil {
			users = append(users, map[string]any{"id": id, "email": email, "name": name, "role": role, "created_at": created})
		}
	}
	s.Page(w, r, "Admin/Users", inertia.Props{"users": users, "flash": flashFromQuery(r)})
}

// adminOrders GET /admin/orders — seluruh order.
func (s *Server) adminOrders(w http.ResponseWriter, r *http.Request, u auth.User) {
	rows, err := s.DB.Query(`SELECT order_no, email, status, subtotal, discount, total, pay_provider, COALESCE(paid_at,''), created_at FROM orders ORDER BY id DESC LIMIT 200`)
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
	s.Page(w, r, "Admin/Orders", inertia.Props{"orders": orders, "flash": flashFromQuery(r)})
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

func (s *Server) adminProducts(w http.ResponseWriter, r *http.Request, u auth.User) {
	rows, err := s.DB.Query(`SELECT id, slug, title, description, price, type, featured, active, file_path, file_name, ext_url,
		(SELECT COUNT(*) FROM license_keys k WHERE k.product_id=p.id AND k.order_item_id IS NULL)
		FROM products p ORDER BY id DESC`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	defer rows.Close()
	var products []map[string]any
	for rows.Next() {
		var id int64
		var slug, title, desc, typ, filePath, fileName, extURL string
		var price, featured, active, keyCount int
		if rows.Scan(&id, &slug, &title, &desc, &price, &typ, &featured, &active, &filePath, &fileName, &extURL, &keyCount) == nil {
			products = append(products, map[string]any{
				"id": id, "slug": slug, "title": title, "description": desc, "price": price,
				"type": typ, "featured": featured == 1, "active": active == 1,
				"file_path": filePath, "file_name": fileName, "ext_url": extURL, "key_count": keyCount,
			})
		}
	}
	s.Page(w, r, "Admin/Products", inertia.Props{"products": products, "images": s.allProductImages(r), "flash": flashFromQuery(r)})
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
		_, err := s.DB.Exec(`INSERT INTO products (slug, title, description, price, type, featured, active, file_path, file_name, ext_url)
			VALUES (?,?,?,?,?,?,?,?,?,?)`,
			slug, title, description, price, ptype, boolInt(featured), boolInt(active), savedPath, savedName, extURL)
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
		s.DB.Exec(`UPDATE products SET slug=?, title=?, description=?, price=?, type=?, featured=?, active=?, file_path=?, file_name=?, ext_url=?, updated_at=datetime('now') WHERE id=?`,
			slug, title, description, price, ptype, boolInt(featured), boolInt(active), savedPath, savedName, extURL, id)
	} else {
		s.DB.Exec(`UPDATE products SET slug=?, title=?, description=?, price=?, type=?, featured=?, active=?, ext_url=?, updated_at=datetime('now') WHERE id=?`,
			slug, title, description, price, ptype, boolInt(featured), boolInt(active), extURL, id)
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
func (s *Server) adminTickets(w http.ResponseWriter, r *http.Request, u auth.User) {
	rows, err := s.DB.Query(`SELECT t.id, t.order_no, t.subject, t.status, t.updated_at, u.email
		FROM tickets t JOIN users u ON u.id = t.user_id ORDER BY t.updated_at DESC LIMIT 200`)
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
	s.Page(w, r, "Admin/Tickets", inertia.Props{"tickets": tickets, "flash": flashFromQuery(r)})
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
			"simulated": pay.IsSimulated(p),
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
