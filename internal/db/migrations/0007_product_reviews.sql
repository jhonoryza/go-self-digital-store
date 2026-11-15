-- Ulasan produk: satu ulasan per item terbeli (order_item_id unik) agar
-- hanya pembeli order lunas yang bisa mengulas, satu kali per unit.
CREATE TABLE product_reviews (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id    INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  order_item_id INTEGER NOT NULL UNIQUE REFERENCES order_items(id) ON DELETE CASCADE,
  rating        INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
  body          TEXT NOT NULL DEFAULT '',
  created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_reviews_product ON product_reviews(product_id);
