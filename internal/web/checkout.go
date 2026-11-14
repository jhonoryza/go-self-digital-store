package web

// Checkout & pembayaran: validasi keranjang server-side, order transaksional,
// tagihan paygateway, halaman bayar ber-token, webhook callback, dan
// pelunasan idempoten.

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	pay "github.com/jhonoryza/paygateway"
)

// publicOrigin origin yang dibuka pembeli (CF Pages bila ada, fallback API).
// Dipakai untuk ReturnURL gateway dan tautan di email.
func (s *Server) publicOrigin(r *http.Request) string {
	if len(s.Cfg.CorsOrigins) > 0 {
		return s.Cfg.CorsOrigins[0]
	}
	return s.baseURL(r)
}

// cartLine item keranjang dari klien (harga divalidasi ulang dari DB).
type cartLine struct {
	ID  int64 `json:"id"`
	Qty int   `json:"qty"`
}

// couponPercent persen diskon kupon aktif; 0 bila tak valid.
func couponPercent(db *sql.DB, code string) int {
	if code == "" {
		return 0
	}
	var pct int
	db.QueryRow(`SELECT percent FROM coupons WHERE code=? AND active=1`, code).Scan(&pct)
	if pct < 1 || pct > 90 {
		return 0
	}
	return pct
}

// checkout POST /checkout — buat order + tagihan. Guest boleh; order terikat
// user bila sedang login.
func (s *Server) checkout(w http.ResponseWriter, r *http.Request) {
	u, _, _ := auth.Resolve(s.DB, r)
	if ipLimited("checkout", s.clientIP(r)) {
		jsonError(w, http.StatusTooManyRequests, "terlalu banyak percobaan")
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	var req struct {
		Items  []cartLine `json:"items"`
		Email  string     `json:"email"`
		Coupon string     `json:"coupon"`
	}
	if err := json.Unmarshal(body, &req); err != nil || len(req.Items) == 0 {
		s.flashErr(w, r, "/keranjang", "Keranjang kosong.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !validEmail(email) {
		s.renderWithErrors(w, r, "Public/Checkout", nil, inertia.ValidationErrors{"email": "email tidak valid - produk dikirim ke email ini"})
		return
	}

	// transaksi: hitung ulang harga dari DB, terapkan kupon, simpan order
	var (
		subtotal int
		lines    []orderItemRow
	)
	for _, it := range req.Items {
		if it.Qty < 1 || it.Qty > 10 {
			s.flashErr(w, r, "/keranjang", "Jumlah tidak valid.")
			return
		}
		var title, ptype string
		var price int
		var active int
		err := s.DB.QueryRow(`SELECT title, type, price, active FROM products WHERE id=?`, it.ID).
			Scan(&title, &ptype, &price, &active)
		if err == sql.ErrNoRows || active != 1 {
			s.flashErr(w, r, "/keranjang", "Produk tidak tersedia: sudah dihapus atau dinonaktifkan.")
			return
		} else if err != nil {
			logErr("checkout produk %d: %v", it.ID, err)
			jsonError(w, http.StatusInternalServerError, "kesalahan server")
			return
		}
		// produk key: satu kunci per unit — qty>1 boleh selama stok cukup (dicek saat lunas)
		subtotal += price * it.Qty
		lines = append(lines, orderItemRow{ProductID: it.ID, Title: title, Price: price, Qty: it.Qty})
	}

	pct := couponPercent(s.DB, strings.ToUpper(strings.TrimSpace(req.Coupon)))
	discount := subtotal * pct / 100
	total := subtotal - discount

	tx, err := s.DB.Begin()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	orderNo := "ORD-" + strings.ToUpper(tok(8))
	dlToken := tok(16)
	var userID any
	if u.ID > 0 {
		userID = u.ID
	}
	_, err = tx.Exec(`INSERT INTO orders (order_no, user_id, email, subtotal, discount, coupon_code, total, dl_token) VALUES (?,?,?,?,?,?,?,?)`,
		orderNo, userID, email, subtotal, discount, strings.ToUpper(strings.TrimSpace(req.Coupon)), total, dlToken)
	if err != nil {
		tx.Rollback()
		logErr("insert order: %v", err)
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	var orderID int64
	tx.QueryRow(`SELECT id FROM orders WHERE order_no=?`, orderNo).Scan(&orderID)
	for _, ln := range lines {
		if _, err := tx.Exec(`INSERT INTO order_items (order_id, product_id, title, price, qty) VALUES (?,?,?,?,?)`,
			orderID, ln.ProductID, ln.Title, ln.Price, ln.Qty); err != nil {
			tx.Rollback()
			logErr("insert item: %v", err)
			jsonError(w, http.StatusInternalServerError, "kesalahan server")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		logErr("commit order: %v", err)
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}

	// tagihan gateway
	payURL, provider, err := createCharge(s.DB, orderNo, total, "Order "+orderNo, email,
		s.baseURL(r)+"/callback/payment", s.publicOrigin(r)+"/pay/"+orderNo+"?t="+dlToken)
	if err != nil {
		logErr("charge %s: %v", orderNo, err)
		s.flashErr(w, r, "/keranjang", "Gateway pembayaran gagal: "+truncate(err.Error(), 120)+" - coba lagi.")
		return
	}
	s.DB.Exec(`UPDATE orders SET pay_url=?, pay_provider=? WHERE id=?`, payURL, provider, orderID)

	// URL pembayaran eksternal (gateway) → location visit Inertia;
	// simulasi/VA (tanpa URL) → halaman bayar internal.
	if payURL != "" {
		s.I.Location(w, r, payURL)
		return
	}
	s.Redirect(w, r, "/pay/"+orderNo+"?t="+dlToken)
}

// orderItemRow snapshot item untuk order_items.
type orderItemRow struct {
	ProductID int64
	Title     string
	Price     int
	Qty       int
}

// orderData payload halaman bayar / detail order.
type orderData struct {
	OrderNo   string        `json:"order_no"`
	Email     string        `json:"email"`
	Status    string        `json:"status"`
	Subtotal  int           `json:"subtotal"`
	Discount  int           `json:"discount"`
	Coupon    string        `json:"coupon_code"`
	Total     int           `json:"total"`
	PayURL    string        `json:"pay_url"`
	PayRef    string        `json:"pay_ref"`
	Provider  string        `json:"pay_provider"`
	CreatedAt string        `json:"created_at"`
	PaidAt    string        `json:"paid_at"`
	Items     []orderOutRow `json:"items"`
}

type orderOutRow struct {
	ID          int64    `json:"id"`
	ProductID   int64    `json:"-"`
	Title       string   `json:"title"`
	Price       int      `json:"price"`
	Qty         int      `json:"qty"`
	Type        string   `json:"type"`
	LicenseKeys []string `json:"license_keys,omitempty"`
	ExtURL      string   `json:"ext_url,omitempty"`
	HasDownload bool     `json:"has_download"`
}

// loadOrder muat order + items (dengan tipe produk untuk delivery).
func (s *Server) loadOrder(orderNo string) (orderData, bool) {
	var o orderData
	var id int64
	err := s.DB.QueryRow(`SELECT id, order_no, email, status, subtotal, discount, coupon_code, total, pay_url, pay_ref, pay_provider, COALESCE(paid_at,'') FROM orders WHERE order_no=?`, orderNo).
		Scan(&id, &o.OrderNo, &o.Email, &o.Status, &o.Subtotal, &o.Discount, &o.Coupon, &o.Total, &o.PayURL, &o.PayRef, &o.Provider, &o.PaidAt)
	if err != nil {
		return o, false
	}
	// catatan: koneksi tunggal SQLite — JANGAN query bersarang saat rows terbuka;
	// kumpulkan items dulu, kunci lisensi di-join sekali setelahnya.
	rows, err := s.DB.Query(`SELECT oi.id, oi.product_id, oi.title, oi.price, oi.qty, p.type, p.ext_url, p.file_path
		FROM order_items oi JOIN products p ON p.id = oi.product_id WHERE oi.order_id=? ORDER BY oi.id`, id)
	if err != nil {
		logErr("loadOrder items %s: %v", orderNo, err)
		return o, true
	}
	keysByItem := map[int64][]string{}
	type itemRaw struct {
		orderOutRow
		extURL   string
		filePath string
	}
	var raws []itemRaw
	for rows.Next() {
		var it itemRaw
		if err := rows.Scan(&it.ID, &it.ProductID, &it.Title, &it.Price, &it.Qty, &it.Type, &it.extURL, &it.filePath); err == nil {
			raws = append(raws, it)
		}
	}
	rows.Close()

	if o.Status == "paid" {
		krows, kerr := s.DB.Query(`SELECT k.order_item_id, k.key FROM license_keys k
			JOIN order_items oi ON oi.id = k.order_item_id WHERE oi.order_id=? ORDER BY k.id`, id)
		if kerr == nil {
			for krows.Next() {
				var iid int64
				var k string
				if krows.Scan(&iid, &k) == nil {
					keysByItem[iid] = append(keysByItem[iid], k)
				}
			}
			krows.Close()
		}
	}

	for _, it := range raws {
		out := it.orderOutRow
		out.HasDownload = out.Type == "file" && it.filePath != ""
		if o.Status == "paid" {
			if out.Type == "link" {
				out.ExtURL = it.extURL
			}
			out.LicenseKeys = keysByItem[out.ID]
		}
		o.Items = append(o.Items, out)
	}
	return o, true
}

// payPage GET /pay/{order_no}?t= — halaman bayar ber-token (anti-enumerasi).
func (s *Server) payPage(w http.ResponseWriter, r *http.Request) {
	o, ok := s.loadOrder(r.PathValue("order_no"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	t := r.URL.Query().Get("t")
	if t == "" || len(t) > 128 {
		http.NotFound(w, r)
		return
	}
	var dlToken string
	s.DB.QueryRow(`SELECT dl_token FROM orders WHERE order_no=?`, o.OrderNo).Scan(&dlToken)
	if subtle.ConstantTimeCompare([]byte(t), []byte(dlToken)) != 1 {
		http.NotFound(w, r)
		return
	}
	_, _, provider := activeProvider(s.DB)
	s.Page(w, r, "Public/Pay", inertia.Props{
		"order":    o,
		"token":    t,
		"simulasi": isSimulated(mustProvider(s.DB)),
		"provider": providerLabel(s.DB, provider),
		"flash":    flashFromQuery(r),
	})
}

// mustProvider ambil provider aktif tanpa cfg (untuk cek simulasi).
func mustProvider(db *sql.DB) pay.Provider {
	p, _, _ := activeProvider(db)
	return p
}

// providerLabel label tampil provider.
func providerLabel(db *sql.DB, name string) string {
	if p, ok := pay.Get(name); ok {
		return p.Label()
	}
	return name
}

// payNow POST /pay/{order_no} — tandai lunas manual, hanya mode simulasi.
func (s *Server) payNow(w http.ResponseWriter, r *http.Request) {
	if !isSimulated(mustProvider(s.DB)) {
		http.NotFound(w, r)
		return
	}
	orderNo := r.PathValue("order_no")
	var dlToken string
	err := s.DB.QueryRow(`SELECT dl_token FROM orders WHERE order_no=?`, orderNo).Scan(&dlToken)
	if err != nil || subtle.ConstantTimeCompare([]byte(r.PostFormValue("t")), []byte(dlToken)) != 1 {
		http.NotFound(w, r)
		return
	}
	s.markOrderPaid(orderNo, r, 0)
	o, _ := s.loadOrder(orderNo)
	target := withParam("/pay/"+orderNo, "t", dlToken)
	s.Redirect(w, r, withParam(target, "ok", statusFlash(o.Status)))
}

func statusFlash(status string) string {
	if status == "paid" {
		return "Pembayaran (simulasi) berhasil - produk terkirim ke email."
	}
	return "Status: " + status
}

// paymentCallback POST /callback/payment — webhook gateway + adapter kustom.
func (s *Server) paymentCallback(w http.ResponseWriter, r *http.Request) {
	if ipLimited("callback", s.clientIP(r)) {
		jsonError(w, http.StatusTooManyRequests, "terlalu banyak permintaan")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// jalur 1: adapter kustom — HMAC-SHA256 body, header X-Signature (opsional, env)
	if s.Cfg.PaymentSecret != "" && r.Header.Get("X-Signature") != "" {
		m := hmac.New(sha256.New, []byte(s.Cfg.PaymentSecret))
		m.Write(body)
		if subtle.ConstantTimeCompare([]byte(hex.EncodeToString(m.Sum(nil))), []byte(r.Header.Get("X-Signature"))) != 1 {
			logErr("callback kustom: signature salah")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var in struct {
			OrderID string `json:"order_id"`
			Status  string `json:"status"`
			Amount  int    `json:"amount"`
		}
		if json.Unmarshal(body, &in) != nil || in.OrderID == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if in.Status == "paid" && s.markOrderPaid(in.OrderID, r, in.Amount) {
			w.Write([]byte(`{"ok":true}`))
			return
		}
		w.Write([]byte(`{"ok":true,"ignored":true}`))
		return
	}

	// jalur 2: provider paygateway aktif
	p, cfg, name := activeProvider(s.DB)
	if isSimulated(p) {
		http.NotFound(w, r)
		return
	}
	cb, err := p.ParseCallback(r.Header, body, cfg)
	if err != nil {
		logErr("callback %s ditolak: %v", name, err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if cb.Ignore || !cb.Paid {
		w.Write([]byte(`{"ok":true,"ignored":true}`))
		return
	}
	if cb.OrderID != "" {
		s.markOrderPaid(cb.OrderID, r, cb.Amount)
	}
	w.Write([]byte(`{"ok":true}`))
}

// markOrderPaid pelunasan idempoten + validasi nominal; true bila pelunasan
// baru terjadi (side effects dijalankan sekali).
func (s *Server) markOrderPaid(orderNo string, r *http.Request, paidAmt int) bool {
	var status string
	var total int
	err := s.DB.QueryRow(`SELECT status, total FROM orders WHERE order_no=?`, orderNo).Scan(&status, &total)
	if err != nil {
		logErr("callback order tak dikenal %q", orderNo)
		return false
	}
	if paidAmt > 0 && paidAmt != total {
		// pembayaran parsial (mis. VA dibayar sebagian) tidak boleh dianggap lunas
		logErr("callback %s: nominal %d ≠ total %d - ditolak", orderNo, paidAmt, total)
		return false
	}
	res, err := s.DB.Exec(`UPDATE orders SET status='paid', paid_at=datetime('now') WHERE order_no=? AND status!='paid'`, orderNo)
	if err != nil {
		logErr("markPaid %s: %v", orderNo, err)
		return false
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return false // sudah lunas sebelumnya - idempoten
	}
	s.onOrderPaid(orderNo, r)
	return true
}
