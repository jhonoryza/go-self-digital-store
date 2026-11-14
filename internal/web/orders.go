package web

// Area user: riwayat pesanan + invoice (order milik sendiri — dicocokkan
// user_id ATAU email akun), dashboard ringkas.

import (
	"net/http"

	inertia "github.com/romsar/gonertia/v3"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
)

// ordersFor muat order milik user (user_id atau email cocok — order guest
// sebelum register tetap muncul bila email sama).
func (s *Server) ordersFor(u auth.User) []orderData {
	rows, err := s.DB.Query(`SELECT order_no, email, status, subtotal, discount, coupon_code, total, pay_provider, COALESCE(paid_at,''), created_at
		FROM orders WHERE user_id=? OR email=? ORDER BY id DESC LIMIT 100`, u.ID, u.Email)
	if err != nil {
		logErr("ordersFor: %v", err)
		return nil
	}
	defer rows.Close()
	var out []orderData
	for rows.Next() {
		var o orderData
		var created string
		if err := rows.Scan(&o.OrderNo, &o.Email, &o.Status, &o.Subtotal, &o.Discount, &o.Coupon, &o.Total, &o.Provider, &o.PaidAt, &created); err == nil {
			out = append(out, o)
		}
	}
	return out
}

// ordersPage GET /pesanan — daftar pesanan milik akun.
func (s *Server) ordersPage(w http.ResponseWriter, r *http.Request, u auth.User) {
	orders := s.ordersFor(u)
	// muat ringkas item per order (judul saja)
	for i := range orders {
		items := s.itemTitles(orders[i].OrderNo)
		orders[i].Items = items
	}
	s.Page(w, r, "Account/Orders", inertia.Props{"orders": orders, "flash": flashFromQuery(r)})
}

// itemTitles ringkas item untuk list.
func (s *Server) itemTitles(orderNo string) []orderOutRow {
	var out []orderOutRow
	rows, err := s.DB.Query(`SELECT oi.title, oi.qty FROM order_items oi
		JOIN orders o ON o.id = oi.order_id WHERE o.order_no=? ORDER BY oi.id`, orderNo)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var it orderOutRow
		if rows.Scan(&it.Title, &it.Qty) == nil {
			out = append(out, it)
		}
	}
	return out
}

// canAccessOrder cek kepemilikan order (user_id atau email akun).
func (s *Server) canAccessOrder(orderNo string, u auth.User) bool {
	var n int
	s.DB.QueryRow(`SELECT COUNT(*) FROM orders WHERE order_no=? AND (user_id=? OR email=?)`, orderNo, u.ID, u.Email).Scan(&n)
	return n > 0
}

// orderDetailPage GET /pesanan/{order_no} — invoice + produk terkirim.
func (s *Server) orderDetailPage(w http.ResponseWriter, r *http.Request, u auth.User) {
	orderNo := r.PathValue("order_no")
	if !s.canAccessOrder(orderNo, u) {
		http.NotFound(w, r)
		return
	}
	o, ok := s.loadOrder(orderNo)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.Page(w, r, "Account/Invoice", inertia.Props{
		"order": o,
		"token": s.orderToken(orderNo),
		"flash": flashFromQuery(r),
	})
}

// ---- dashboard ringkas ----

func (s *Server) dashboardData(u auth.User) map[string]any {
	var paidCount, pendingCount, spend int
	s.DB.QueryRow(`SELECT COUNT(*) FROM orders WHERE (user_id=? OR email=?) AND status='paid'`, u.ID, u.Email).Scan(&paidCount)
	s.DB.QueryRow(`SELECT COUNT(*) FROM orders WHERE (user_id=? OR email=?) AND status='pending'`, u.ID, u.Email).Scan(&pendingCount)
	s.DB.QueryRow(`SELECT COALESCE(SUM(total),0) FROM orders WHERE (user_id=? OR email=?) AND status='paid'`, u.ID, u.Email).Scan(&spend)

	// pesanan terakhir
	orders := s.ordersFor(u)
	if len(orders) > 5 {
		orders = orders[:5]
	}
	for i := range orders {
		orders[i].Items = s.itemTitles(orders[i].OrderNo)
	}
	return map[string]any{
		"paid_count":    paidCount,
		"pending_count": pendingCount,
		"total_spend":   spend,
		"recent":        orders,
	}
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request, u auth.User) {
	s.Page(w, r, "Account/Dashboard", inertia.Props{
		"stats": s.dashboardData(u),
		"flash": flashFromQuery(r),
	})
}
