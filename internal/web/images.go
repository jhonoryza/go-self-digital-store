package web

// Gambar produk: disimpan di bucket Cloudflare R2 bersama ("webapp") dengan
// prefix per-app agar tak bertabrakan dengan app lain. Publiknya lewat proxy
// GET /images/{key...} di API (atau domain publik bucket bila diisi
// STORE_R2_PUBLIC_BASE). Maks gambar per produk: config.R2.MaxImages (5).

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/jhonoryza/go-self-digital-store/internal/auth"
	"github.com/jhonoryza/go-self-digital-store/internal/config"
)

// imageStore abstraksi objek storage — diganti stub pada test.
type imageStore interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Delete(ctx context.Context, key string) error
	Get(ctx context.Context, key string) (contentType string, data []byte, err error)
}

// r2Store imageStore di atas bucket R2 (S3 API via minio-go).
type r2Store struct {
	client *minio.Client
	bucket string
}

// newR2Store bangun klien S3 ke R2; nil bila config belum lengkap.
func newR2Store(cfg config.R2) *r2Store {
	if !cfg.Enabled() {
		return nil
	}
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: strings.HasPrefix(cfg.Endpoint, "https://"),
	})
	if err != nil {
		logErr("r2 client: %v", err)
		return nil
	}
	bucket := cfg.Bucket
	if bucket == "" {
		bucket = "webapp"
	}
	return &r2Store{client: client, bucket: bucket}
}

func (s *r2Store) Put(ctx context.Context, key, contentType string, data []byte) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *r2Store) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *r2Store) Get(ctx context.Context, key string) (string, []byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return "", nil, err
	}
	defer obj.Close()
	st, err := obj.Stat()
	if err != nil {
		return "", nil, err
	}
	data, err := io.ReadAll(obj)
	if err != nil {
		return "", nil, err
	}
	return st.ContentType, data, nil
}

// r2Prefix prefix path per-app di bucket bersama; default bila config literal
// kosong (test).
func (s *Server) r2Prefix() string {
	if p := s.Cfg.R2.Prefix; p != "" {
		return p
	}
	return "go-self-digital-store"
}

// productImageRow gambar untuk props frontend.
type productImageRow struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

// imageURL URL publik satu objek gambar: domain publik bucket bila diisi,
// selain itu proxy API (origin yang sama dengan halaman dev).
func (s *Server) imageURL(r *http.Request, objectKey string) string {
	if s.Cfg.R2.PublicBase != "" {
		return s.Cfg.R2.PublicBase + "/" + objectKey
	}
	return s.baseURL(r) + "/images/" + objectKey
}

// productImagesOf semua gambar satu produk, urut sort; URL absolut per request.
func (s *Server) productImagesOf(r *http.Request, productID int64) []productImageRow {
	rows, err := s.DB.Query(`SELECT id, object_key FROM product_images WHERE product_id=? ORDER BY sort, id`, productID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []productImageRow{} // array kosong, bukan nil - prop wajib ada
	for rows.Next() {
		var id int64
		var key string
		if rows.Scan(&id, &key) == nil {
			out = append(out, productImageRow{ID: id, URL: s.imageURL(r, key)})
		}
	}
	return out
}

// allProductImages seluruh gambar dikelompokkan per produk (konsol admin).
func (s *Server) allProductImages(r *http.Request) map[int64][]productImageRow {
	out := map[int64][]productImageRow{}
	rows, err := s.DB.Query(`SELECT product_id, id, object_key FROM product_images ORDER BY product_id, sort, id`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var pid, id int64
		var key string
		if rows.Scan(&pid, &id, &key) == nil {
			out[pid] = append(out[pid], productImageRow{ID: id, URL: s.imageURL(r, key)})
		}
	}
	return out
}

// firstImageURLs satu gambar pertama untuk banyak produk sekaligus (kartu
// katalog) — hindari N+1; kembalikan map product_id → object_key.
func (s *Server) firstImageURLs(productIDs []int64) map[int64]string {
	out := map[int64]string{}
	if len(productIDs) == 0 {
		return out
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(productIDs)), ",")
	args := make([]any, len(productIDs))
	for i, id := range productIDs {
		args[i] = id
	}
	rows, err := s.DB.Query(`SELECT pi.product_id, pi.object_key FROM product_images pi
		JOIN (SELECT product_id, MIN(sort) AS first_sort FROM product_images GROUP BY product_id) f
		  ON f.product_id = pi.product_id AND f.first_sort = pi.sort
		WHERE pi.product_id IN (`+placeholders+`)`, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var key string
		if rows.Scan(&id, &key) == nil {
			out[id] = key
		}
	}
	return out
}

// attachFirstImages isi Product.ImageURL untuk daftar produk (kartu katalog).
func (s *Server) attachFirstImages(r *http.Request, products []productData) {
	ids := make([]int64, 0, len(products))
	for _, p := range products {
		ids = append(ids, p.ID)
	}
	for id, key := range s.firstImageURLs(ids) {
		for i := range products {
			if products[i].ID == id {
				products[i].ImageURL = s.imageURL(r, key)
			}
		}
	}
}

// allowedImageTypes tipe konten gambar yang diterima + ekstensi simpannya.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

const maxImageBytes = 2 << 20 // 2 MB per gambar

// adminProductImages POST /admin/produk/{id}/images — multipart field
// "images" (bisa banyak); tolak bila total melebihi maks per produk.
func (s *Server) adminProductImages(w http.ResponseWriter, r *http.Request, u auth.User) {
	if s.Images == nil {
		s.flashErr(w, r, "/admin/produk", "Penyimpanan gambar belum dikonfigurasi (STORE_R2_*).")
		return
	}
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

	if err := r.ParseMultipartForm(12 << 20); err != nil {
		s.flashErr(w, r, "/admin/produk", "Unggahan terlalu besar (maks total 12 MB).")
		return
	}
	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		s.flashErr(w, r, "/admin/produk", "Tidak ada gambar dipilih.")
		return
	}
	maxImages := s.Cfg.R2.MaxImages
	if maxImages < 1 {
		maxImages = 5 // konfigurasi literal tanpa R2 tetap dapat default yang wajar
	}
	var current int
	s.DB.QueryRow(`SELECT COUNT(*) FROM product_images WHERE product_id=?`, id).Scan(&current)
	if current+len(files) > maxImages {
		s.flashErr(w, r, "/admin/produk",
			fmt.Sprintf("Maks %d gambar per produk (ada %d, unggah %d).", maxImages, current, len(files)))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	saved := 0
	for _, fh := range files {
		if fh.Size > maxImageBytes {
			s.flashErr(w, r, "/admin/produk", fmt.Sprintf("%s: ukuran melebihi 2 MB.", fh.Filename))
			return
		}
		f, err := fh.Open()
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "kesalahan server")
			return
		}
		data, err := io.ReadAll(f) // baca penuh untuk sniffing tipe asli
		f.Close()
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "kesalahan server")
			return
		}
		sniffed := http.DetectContentType(data)
		ext, ok := allowedImageTypes[sniffed]
		if !ok {
			s.flashErr(w, r, "/admin/produk", fmt.Sprintf("%s: tipe %s tidak didukung (jpg/png/webp).", fh.Filename, sniffed))
			return
		}
		key := path.Join(s.r2Prefix(), "products", fmtInt(id), fmt.Sprintf("%s%s", randomHex(8), ext))
		if err := s.Images.Put(ctx, key, sniffed, data); err != nil {
			logErr("upload gambar %s: %v", key, err)
			s.flashErr(w, r, "/admin/produk", "Gagal mengunggah ke penyimpanan — coba lagi.")
			return
		}
		var nextSort int
		s.DB.QueryRow(`SELECT COALESCE(MAX(sort),0)+1 FROM product_images WHERE product_id=?`, id).Scan(&nextSort)
		if _, err := s.DB.Exec(`INSERT INTO product_images (product_id, object_key, sort) VALUES (?,?,?)`, id, key, nextSort); err != nil {
			s.Images.Delete(ctx, key) // jangan sisakan objek yatim
			logErr("insert product_images: %v", err)
			jsonError(w, http.StatusInternalServerError, "kesalahan server")
			return
		}
		saved++
	}
	audit(s.DB, "produk.gambar", slug, fmt.Sprintf("+%d oleh %s", saved, u.Email))
	s.flashOK(w, r, "/admin/produk", fmt.Sprintf("%d gambar ditambahkan.", saved))
}

// adminProductImageDelete POST /admin/produk/{id}/images/{image_id}/delete.
func (s *Server) adminProductImageDelete(w http.ResponseWriter, r *http.Request, u auth.User) {
	if s.Images == nil {
		http.NotFound(w, r)
		return
	}
	id, err := pathInt(r.PathValue("id"))
	imgID, err2 := pathInt(r.PathValue("image_id"))
	if err != nil || err2 != nil {
		http.NotFound(w, r)
		return
	}
	var key string
	if err := s.DB.QueryRow(`SELECT object_key FROM product_images WHERE id=? AND product_id=?`, imgID, id).Scan(&key); err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.Images.Delete(r.Context(), key); err != nil {
		logErr("hapus objek %s: %v", key, err) // lanjut hapus baris walau objek gagal
	}
	if _, err := s.DB.Exec(`DELETE FROM product_images WHERE id=?`, imgID); err != nil {
		jsonError(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	audit(s.DB, "produk.gambar_hapus", fmtInt(id), key+" oleh "+u.Email)
	s.flashOK(w, r, "/admin/produk", "Gambar dihapus.")
}

// productImageProxy GET /images/{key...} — alirkan objek gambar dari R2;
// hanya key di bawah prefix app yang boleh (bukan seluruh bucket).
func (s *Server) productImageProxy(w http.ResponseWriter, r *http.Request) {
	if s.Images == nil {
		http.NotFound(w, r)
		return
	}
	key := r.PathValue("key")
	if key == "" || !strings.HasPrefix(key, s.r2Prefix()+"/") || strings.Contains(key, "..") {
		http.NotFound(w, r)
		return
	}
	contentType, data, err := s.Images.Get(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	// nama objek acak per unggahan → konten tak berubah, cache seumur hidup
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31556952, immutable")
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	w.Write(data)
}

// randomHex n byte acak hex untuk nama objek.
func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
