-- SEO opsional per produk untuk meta tag halaman detail.
ALTER TABLE products ADD COLUMN meta_title TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN meta_description TEXT NOT NULL DEFAULT '';
