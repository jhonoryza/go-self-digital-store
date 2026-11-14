-- Gambar produk: di-host di bucket R2 bersama (prefix per-app), maks 5 per
-- produk; object_key adalah path objek di bucket (tanpa nama bucket).

CREATE TABLE product_images (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  object_key TEXT NOT NULL,
  sort       INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_product_images_product ON product_images(product_id, sort);
