package web

// Handler halaman publik: landing, daftar & detail produk, health.

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	inertia "github.com/romsar/gonertia/v3"
)

// productData ringkasan produk untuk prop Inertia.
type productData struct {
	ID          int64    `json:"id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Price       int      `json:"price"`
	Type        string   `json:"type"`
	Featured    bool     `json:"featured"`
	Category    string   `json:"category,omitempty"`  // kategori opsional
	Tags        []string `json:"tags,omitempty"`      // tag opsional (JSON di DB)
	ImageURL    string   `json:"image_url,omitempty"` // gambar pertama (kartu katalog)
	MetaTitle   string   `json:"meta_title,omitempty"`
	MetaDesc    string   `json:"meta_description,omitempty"`
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

// parseTagsJSON ubah kolom tags (JSON array) menjadi slice; rusak = kosong.
func parseTagsJSON(raw string) []string {
	if raw == "" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil
	}
	return tags
}

// scanCatalog seperti scanProducts + kolom category & tags (9 kolom).
func scanCatalog(rows *sql.Rows, err error) []productData {
	if err != nil {
		logErr("query products: %v", err)
		return nil
	}
	defer rows.Close()
	var out []productData
	for rows.Next() {
		var p productData
		var featured int
		var tagsJSON string
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Description, &p.Price, &p.Type, &featured, &p.Category, &tagsJSON); err == nil {
			p.Featured = featured == 1
			p.Tags = parseTagsJSON(tagsJSON)
			out = append(out, p)
		}
	}
	return out
}

// landing hero + produk unggulan.
func (s *Server) landing(w http.ResponseWriter, r *http.Request) {
	featured := scanCatalog(s.DB.Query(
		`SELECT id, slug, title, description, price, type, featured, category, tags FROM products WHERE active=1 AND featured=1 ORDER BY id DESC LIMIT 6`))
	s.attachFirstImages(r, featured)
	s.Page(w, r, "Public/Landing", inertia.Props{
		"featured": featured,
		"seo":      s.seoFor("seo.home.title", "seo.home.description"),
		"flash":    flashFromQuery(r),
	})
}

// productsPerPage jumlah produk aktif per halaman katalog.
const productsPerPage = 12

// catalogFilter parameter katalog dari query string: pencarian nama, kategori,
// tag, rentang harga, dan urutan. Nilai tidak valid diabaikan (bukan error).
type catalogFilter struct {
	Q        string `json:"q"`
	Kategori string `json:"kategori"`
	Tag      string `json:"tag"`
	HargaMin int    `json:"harga_min"`
	HargaMax int    `json:"harga_max"`
	Sort     string `json:"sort"`
}

// catalogSorts mapping sort -> ORDER BY (id sebagai pemecah seri).
var catalogSorts = map[string]string{
	"termurah": "price ASC, id DESC",
	"termahal": "price DESC, id DESC",
	"terbaru":  "created_at DESC, id DESC",
	"terlama":  "created_at ASC, id ASC",
}

// parseCatalogFilter baca parameter katalog dari query string.
func parseCatalogFilter(r *http.Request) catalogFilter {
	f := catalogFilter{
		Q:        strings.TrimSpace(r.URL.Query().Get("q")),
		Kategori: strings.TrimSpace(r.URL.Query().Get("kategori")),
		Tag:      strings.TrimSpace(r.URL.Query().Get("tag")),
		Sort:     r.URL.Query().Get("sort"),
	}
	f.Tag = strings.ReplaceAll(f.Tag, `"`, "") // tag dicocokkan sebagai token JSON
	if _, ok := catalogSorts[f.Sort]; !ok {
		f.Sort = "terbaru"
	}
	f.HargaMin, _ = strconv.Atoi(r.URL.Query().Get("harga_min"))
	f.HargaMax, _ = strconv.Atoi(r.URL.Query().Get("harga_max"))
	return f
}

// where bangun kondisi WHERE (+args) dari filter; kosong berarti tanpa filter.
func (f catalogFilter) where() (string, []any) {
	var conds []string
	var args []any
	if f.Q != "" {
		esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(f.Q)
		conds = append(conds, `title LIKE ? ESCAPE '\'`)
		args = append(args, "%"+esc+"%")
	}
	if f.Kategori != "" {
		conds = append(conds, `LOWER(category)=LOWER(?)`)
		args = append(args, f.Kategori)
	}
	if f.Tag != "" {
		conds = append(conds, `tags LIKE ?`)
		args = append(args, `%"`+f.Tag+`"%`)
	}
	if f.HargaMin > 0 {
		conds = append(conds, `price>=?`)
		args = append(args, f.HargaMin)
	}
	if f.HargaMax > 0 {
		conds = append(conds, `price<=?`)
		args = append(args, f.HargaMax)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(conds, " AND "), args
}

// catalogCategories kategori unik produk aktif (urut abjad) untuk dropdown.
func (s *Server) catalogCategories() []string {
	rows, err := s.DB.Query(`SELECT DISTINCT category FROM products WHERE active=1 AND category != '' ORDER BY category`)
	if err != nil {
		logErr("query kategori: %v", err)
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if rows.Scan(&c) == nil && c != "" {
			out = append(out, c)
		}
	}
	return out
}

// catalogTags tag unik hasil parse kolom JSON tags produk aktif.
func (s *Server) catalogTags() []string {
	rows, err := s.DB.Query(`SELECT tags FROM products WHERE active=1 AND tags NOT IN ('', '[]')`)
	if err != nil {
		logErr("query tags: %v", err)
		return nil
	}
	defer rows.Close()
	seen := map[string]bool{}
	var out []string
	for rows.Next() {
		var raw string
		if rows.Scan(&raw) != nil {
			continue
		}
		for _, t := range parseTagsJSON(raw) {
			if t != "" && !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	sort.Strings(out)
	return out
}

// productsList katalog produk aktif: paginasi 12/halaman + pencarian nama,
// filter kategori/tag/rentang harga, dan urutan (?q&kategori&tag&harga_min&harga_max&sort&page).
func (s *Server) productsList(w http.ResponseWriter, r *http.Request) {
	f := parseCatalogFilter(r)
	where, args := f.where()

	var total int
	s.DB.QueryRow(`SELECT COUNT(*) FROM products WHERE active=1`+where, args...).Scan(&total)
	lastPage := (total + productsPerPage - 1) / productsPerPage
	if lastPage < 1 {
		lastPage = 1
	}
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p >= 1 && p <= lastPage {
		page = p
	}

	// daftar pilihan filter (query terpisah, tidak bersarang dengan rows)
	categories := s.catalogCategories()
	tags := s.catalogTags()

	q := `SELECT id, slug, title, description, price, type, featured, category, tags FROM products WHERE active=1` +
		where + ` ORDER BY ` + catalogSorts[f.Sort] + ` LIMIT ? OFFSET ?`
	qargs := append(append([]any{}, args...), productsPerPage, (page-1)*productsPerPage)
	products := scanCatalog(s.DB.Query(q, qargs...))
	s.attachFirstImages(r, products)
	s.Page(w, r, "Public/Products", inertia.Props{
		"products":   products,
		"page":       page,
		"last_page":  lastPage,
		"total":      total,
		"categories": categories,
		"tags":       tags,
		"filters":    f,
		"seo":        s.seoFor("seo.products.title", "seo.products.description"),
		"flash":      flashFromQuery(r),
	})
}

// productDetail satu produk aktif; 404 bila tidak ada.
func (s *Server) productDetail(w http.ResponseWriter, r *http.Request) {
	var p productData
	var tagsJSON string
	err := s.DB.QueryRow(
		`SELECT id, slug, title, description, price, type, featured, meta_title, meta_description, category, tags FROM products WHERE slug=? AND active=1`, r.PathValue("slug")).
		Scan(&p.ID, &p.Slug, &p.Title, &p.Description, &p.Price, &p.Type, &p.Featured, &p.MetaTitle, &p.MetaDesc, &p.Category, &tagsJSON)
	p.Tags = parseTagsJSON(tagsJSON)
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
		"images":  s.productImagesOf(r, p.ID),
		"seo":     s.seoFor("seo.product.title", "seo.product.description"),
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
