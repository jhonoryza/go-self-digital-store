package web

// Handler halaman publik: landing, daftar & detail produk, health.

import (
	"database/sql"
	"net/http"

	inertia "github.com/romsar/gonertia/v3"
)

// productData ringkasan produk untuk prop Inertia.
type productData struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Price       int    `json:"price"`
	Type        string `json:"type"`
	Featured    bool   `json:"featured"`
}

func scanProducts(rows *sql.Rows, err error) []productData {
	if err != nil {
		logErr("query products: %v", err)
		return nil
	}
	defer rows.Close()
	var out []productData
	for rows.Next() {
		var p productData
		var featured int
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Description, &p.Price, &p.Type, &featured); err == nil {
			p.Featured = featured == 1
			out = append(out, p)
		}
	}
	return out
}

// landing hero + produk unggulan.
func (s *Server) landing(w http.ResponseWriter, r *http.Request) {
	featured := scanProducts(s.DB.Query(
		`SELECT id, slug, title, description, price, type, featured FROM products WHERE active=1 AND featured=1 ORDER BY id DESC LIMIT 6`))
	s.Page(w, r, "Public/Landing", inertia.Props{
		"featured": featured,
		"flash":    flashFromQuery(r),
	})
}

// productsList semua produk aktif.
func (s *Server) productsList(w http.ResponseWriter, r *http.Request) {
	products := scanProducts(s.DB.Query(
		`SELECT id, slug, title, description, price, type, featured FROM products WHERE active=1 ORDER BY id DESC`))
	s.Page(w, r, "Public/Products", inertia.Props{
		"products": products,
		"flash":    flashFromQuery(r),
	})
}

// productDetail satu produk aktif; 404 bila tidak ada.
func (s *Server) productDetail(w http.ResponseWriter, r *http.Request) {
	var p productData
	err := s.DB.QueryRow(
		`SELECT id, slug, title, description, price, type, featured FROM products WHERE slug=? AND active=1`, r.PathValue("slug")).
		Scan(&p.ID, &p.Slug, &p.Title, &p.Description, &p.Price, &p.Type, &p.Featured)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		logErr("produk %s: %v", r.PathValue("slug"), err)
		http.Error(w, "kesalahan server", http.StatusInternalServerError)
		return
	}
	s.Page(w, r, "Public/ProductDetail", inertia.Props{
		"product": p,
		"flash":   flashFromQuery(r),
	})
}

// healthz probe hidup + DB.
func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := s.DB.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"ok":false}`))
		return
	}
	w.Write([]byte(`{"ok":true}`))
}

// flashFromQuery baca ?ok=/&err= (flash ringan lintas redirect).
func flashFromQuery(r *http.Request) map[string]string {
	f := map[string]string{}
	if v := r.URL.Query().Get("ok"); v != "" {
		f["ok"] = v
	}
	if v := r.URL.Query().Get("err"); v != "" {
		f["err"] = v
	}
	return f
}
