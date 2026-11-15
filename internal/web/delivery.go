package web

// Delivery: saat order lunas — assign kunci lisensi (type=key), kirim email
// produk (unduhan/kunci/tautan), dan sajikan unduhan ber-token (type=file).
// Plus drip reminder order pending.

import (
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jhonoryza/go-self-digital-store/internal/mail"
)

// filesDir direktori berkas privat (tidak pernah dilayani publik langsung).
func (s *Server) filesDir() string { return filepath.Join(s.Cfg.Dir, "files") }

// orderToken token unduhan/halaman bayar order.
func (s *Server) orderToken(orderNo string) string {
	var t string
	s.DB.QueryRow(`SELECT dl_token FROM orders WHERE order_no=?`, orderNo).Scan(&t)
	return t
}

// onOrderPaid efek samping pelunasan: assign kunci lisensi lalu kirim email
// produk (dijalankan sekali — dipanggil dari markOrderPaid setelah guard
// RowsAffected).
func (s *Server) onOrderPaid(orderNo string, r *http.Request) {
	o, ok := s.loadOrder(orderNo)
	if !ok {
		return
	}

	// 1. assign kunci lisensi: satu kunci per unit, dari pool stok admin.
	// Checkout sudah memvalidasi stok, tapi order lain bisa membayar lebih
	// dulu — kekurangan dicatat ke audit agar admin mengisi pool.
	shortfall := []string{}
	for _, it := range o.Items {
		if it.Type != "key" {
			continue
		}
		terpasang := 0
		for q := 0; q < it.Qty; q++ {
			var key string
			err := s.DB.QueryRow(`UPDATE license_keys SET order_item_id=?
				WHERE id=(SELECT id FROM license_keys WHERE product_id=? AND order_item_id IS NULL ORDER BY id LIMIT 1)
				RETURNING key`, it.ID, it.ProductID).Scan(&key)
			if err != nil {
				logErr("stok kunci habis utk item %d (order %s): %v", it.ID, orderNo, err)
				break
			}
			terpasang++
		}
		if terpasang < it.Qty {
			shortfall = append(shortfall, fmt.Sprintf("%s: %d/%d kunci", it.Title, terpasang, it.Qty))
		}
	}
	if len(shortfall) > 0 {
		audit(s.DB, "order.key_shortfall", orderNo, strings.Join(shortfall, "; "))
	}
	o, _ = s.loadOrder(orderNo) // muat ulang agar kunci ter-assign ikut

	// 2. email delivery (async)
	origin := s.publicOrigin(r)
	dlToken := s.orderToken(orderNo)
	order := o
	go func() {
		var b strings.Builder
		fmt.Fprintf(&b, `<p>Halo,</p><p>Terima kasih! Order <strong>%s</strong> sudah lunas. Berikut produk Anda:</p>`, order.OrderNo)
		b.WriteString(`<div style="border:1px solid #e4e4e7;border-radius:8px;padding:16px;margin:16px 0;">`)
		for _, it := range order.Items {
			fmt.Fprintf(&b, `<p style="margin:0 0 8px;"><strong>%s</strong>%s<br>`, it.Title, qtySuffix(it.Qty))
			switch {
			case it.HasDownload:
				link := fmt.Sprintf("%s/dl/%s/%d?t=%s", origin, order.OrderNo, it.ID, dlToken)
				b.WriteString(mail.Button(link, "Unduh berkas"))
			case it.Type == "key":
				for _, k := range it.LicenseKeys {
					fmt.Fprintf(&b, `<br><span style="font-family:monospace;background:#f4f4f5;padding:2px 6px;border-radius:4px;">%s</span>`, k)
				}
				if len(it.LicenseKeys) == 0 {
					b.WriteString(`<br><em style="color:#a1a1aa;">Kunci akan dikirim menyusul (stok diisi admin).</em>`)
				}
			case it.Type == "link":
				b.WriteString(mail.Button(it.ExtURL, "Buka produk"))
			}
			b.WriteString(`</p>`)
		}
		b.WriteString(`</div>`)
		b.WriteString(`<p>Halaman order: ` + mail.Button(origin+"/pay/"+order.OrderNo+"?t="+dlToken, "Lihat order") + `</p>`)
		body := mail.Style(s.appName(), b.String())
		s.Mail.Send(order.Email, "Produk Anda dari "+s.appName()+" ("+order.OrderNo+")", body)
	}()
}

func qtySuffix(qty int) string {
	if qty > 1 {
		return fmt.Sprintf(" (×%d)", qty)
	}
	return ""
}

// filesPrefix prefix object key berkas produk di bucket R2.
const filesPrefix = "files/"

// downloadPresignTTL umur presigned URL unduhan.
const downloadPresignTTL = 15 * time.Minute

// putProductFile unggah berkas produk ke R2 (key files/<name>); false bila
// R2 tidak tersedia atau gagal - pemanggil fallback menyimpan ke disk lokal.
func (s *Server) putProductFile(name, contentType string, file io.Reader) bool {
	if s.Images == nil {
		return false
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return false
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return s.Images.Put(ctx, filesPrefix+name, contentType, data) == nil
}

// hapusProductFile bersihkan berkas produk dari R2 dan/atau disk lokal
// (berkas lama mungkin hanya ada di salah satu).
func (s *Server) hapusProductFile(path string) {
	if path == "" {
		return
	}
	if s.Images != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_ = s.Images.Delete(ctx, filesPrefix+path)
		cancel()
	}
	_ = os.Remove(filepath.Join(s.filesDir(), filepath.Clean(path)))
}

// download GET /dl/{order_no}/{item_id}?t= — unduhan ber-token (order lunas).
func (s *Server) download(w http.ResponseWriter, r *http.Request) {
	if ipLimited("download", s.clientIP(r)) {
		http.Error(w, "terlalu banyak permintaan", http.StatusTooManyRequests)
		return
	}
	orderNo := r.PathValue("order_no")
	itemID := r.PathValue("item")
	t := r.URL.Query().Get("t")

	var dlToken, status string
	err := s.DB.QueryRow(`SELECT dl_token, status FROM orders WHERE order_no=?`, orderNo).Scan(&dlToken, &status)
	if err != nil || status != "paid" || subtle.ConstantTimeCompare([]byte(t), []byte(dlToken)) != 1 {
		http.Error(w, "tautan tidak berlaku", http.StatusForbidden)
		return
	}
	var filePath, fileName string
	err = s.DB.QueryRow(`SELECT p.file_path, p.file_name FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		WHERE oi.id=? AND oi.order_id=(SELECT id FROM orders WHERE order_no=?) AND p.type='file'`, itemID, orderNo).
		Scan(&filePath, &fileName)
	if err != nil || filePath == "" {
		http.NotFound(w, r)
		return
	}
	if fileName == "" {
		fileName = filepath.Base(filePath)
	}
	// utamakan presigned URL R2: TTL 15 menit dan tautan selalu berubah
	if r2s, ok := s.Images.(*r2Store); ok {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		u, err := r2s.PresignGet(ctx, filesPrefix+filePath, downloadPresignTTL, fileName)
		cancel()
		if err == nil {
			http.Redirect(w, r, u, http.StatusFound)
			return
		}
		logErr("presign %s/%s: %v - fallback disk lokal", orderNo, itemID, err)
	}
	f, err := os.Open(filepath.Join(s.filesDir(), filepath.Clean(filePath)))
	if err != nil {
		logErr("unduhan %s/%s: %v", orderNo, itemID, err)
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	http.ServeContent(w, r, fileName, time.Now(), f)
}

// remindPending kirim pengingat sekali untuk order pending > 1 jam.
func (s *Server) remindPending(r *http.Request) {
	rows, err := s.DB.Query(`SELECT order_no, email, total, dl_token FROM orders
		WHERE status='pending' AND reminded=0 AND created_at < datetime('now', '-1 hour')`)
	if err != nil {
		return
	}
	type pendingOrder struct {
		no, email string
		total     int
		token     string
	}
	var list []pendingOrder
	for rows.Next() {
		var p pendingOrder
		if rows.Scan(&p.no, &p.email, &p.total, &p.token) == nil {
			list = append(list, p)
		}
	}
	rows.Close()
	for _, p := range list {
		link := s.publicOrigin(r) + "/pay/" + p.no + "?t=" + p.token
		body := mail.Style(s.appName(),
			fmt.Sprintf(`<p>Halo,</p><p>Order <strong>%s</strong> (total <strong>%s</strong>) masih menunggu pembayaran.</p><p>%s</p><p style="color:#71717a;">Pengingat ini dikirim sekali; halaman order tetap bisa diakses dari tautan di email order awal.</p>`,
				p.no, rupiah(p.total), mail.Button(link, "Selesaikan pembayaran")))
		s.Mail.Send(p.email, "Selesaikan pembayaran order "+p.no, body)
		s.DB.Exec(`UPDATE orders SET reminded=1 WHERE order_no=?`, p.no)
	}
}

// Maintain pekerja berkala: pengingat order pending (dipanggil loop main).
func (s *Server) Maintain() {
	if !s.Mail.Enabled() {
		return
	}
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	s.remindPending(r)
}
