package web

// Ulasan produk: hanya pembeli order LUNAS yang bisa mengulas, satu ulasan
// per item terbeli (order_item_id unik). Rating 1-5 + teks opsional.

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
)

// reviewData satu ulasan untuk props.
type reviewData struct {
	Rating    int    `json:"rating"`
	Body      string `json:"body"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// productReviews muat ulasan + ringkasan rata-rata satu produk.
func (s *Server) productReviews(productID int64) ([]reviewData, float64, int) {
	reviews := []reviewData{}
	rows, err := s.DB.Query(`SELECT r.rating, r.body, r.created_at, COALESCE(NULLIF(u.name,''), u.email)
		FROM product_reviews r JOIN users u ON u.id = r.user_id
		WHERE r.product_id=? ORDER BY r.id DESC LIMIT 20`, productID)
	if err == nil {
		for rows.Next() {
			var rv reviewData
			if rows.Scan(&rv.Rating, &rv.Body, &rv.CreatedAt, &rv.Name) == nil {
				reviews = append(reviews, rv)
			}
		}
		rows.Close()
	}
	var avg, count float64
	s.DB.QueryRow(`SELECT COALESCE(AVG(rating),0), COUNT(*) FROM product_reviews WHERE product_id=?`, productID).Scan(&avg, &count)
	return reviews, avg, int(count)
}

// reviewEntitlement cari order_item lunas milik user yang belum diulas utk
// produk ini. Mengembalikan (order_item_id, true).
func (s *Server) reviewEntitlement(productID, userID int64, email string) (int64, bool) {
	var itemID int64
	err := s.DB.QueryRow(`SELECT oi.id FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE oi.product_id=? AND o.status='paid' AND (o.user_id=? OR o.email=?)
		  AND oi.id NOT IN (SELECT order_item_id FROM product_reviews WHERE user_id=?)
		ORDER BY oi.id LIMIT 1`, productID, userID, email, userID).Scan(&itemID)
	return itemID, err == nil
}

// reviewStore POST /produk/{slug}/ulasan — simpan ulasan pembeli.
func (s *Server) reviewStore(w http.ResponseWriter, r *http.Request, u auth.User) {
	var productID int64
	err := s.DB.QueryRow(`SELECT id FROM products WHERE slug=? AND active=1`, r.PathValue("slug")).Scan(&productID)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}

	rating, _ := strconv.Atoi(strings.TrimSpace(r.PostFormValue("rating")))
	body := strings.TrimSpace(r.PostFormValue("body"))
	if rating < 1 || rating > 5 {
		s.renderWithErrors(w, r, "Public/ProductDetail", nil,
			inertia.ValidationErrors{"rating": "Pilih rating 1 sampai 5 bintang."})
		return
	}
	if len(body) > 1000 {
		body = body[:1000]
	}

	itemID, ok := s.reviewEntitlement(productID, u.ID, u.Email)
	if !ok {
		s.flashErr(w, r, "/produk/"+r.PathValue("slug"), "Ulasan hanya untuk pembeli dengan order lunas yang belum mengulas.")
		return
	}
	res, err := s.DB.Exec(`INSERT INTO product_reviews (product_id, user_id, order_item_id, rating, body) VALUES (?,?,?,?,?)`,
		productID, u.ID, itemID, rating, body)
	if err != nil {
		s.flashErr(w, r, "/produk/"+r.PathValue("slug"), "Ulasan gagal disimpan.")
		return
	}
	if n, _ := res.RowsAffected(); n == 1 {
		audit(s.DB, "ulasan.buat", r.PathValue("slug"), fmt.Sprintf("%d bintang oleh %s", rating, u.Email))
	}
	s.flashOK(w, r, "/produk/"+r.PathValue("slug"), "Terima kasih atas ulasannya!")
}
