-- Skema awal: users, sesi, reset password, katalog produk digital, order,
-- tiket, mail log, settings, kupon, audit log.

CREATE TABLE users (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  email      TEXT NOT NULL UNIQUE,
  name       TEXT NOT NULL DEFAULT '',
  password   TEXT NOT NULL,
  role       TEXT NOT NULL DEFAULT 'user', -- user | admin
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE sessions (
  token      TEXT PRIMARY KEY, -- sha256 hex dari token cookie
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  csrf_token TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE password_resets (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  email      TEXT NOT NULL,
  token      TEXT NOT NULL UNIQUE,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE products (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  slug        TEXT NOT NULL UNIQUE,
  title       TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  price       INTEGER NOT NULL,            -- rupiah
  type        TEXT NOT NULL DEFAULT 'file', -- file | key | link
  file_path   TEXT NOT NULL DEFAULT '',    -- relatif di bawah dir files/ (type=file)
  file_name   TEXT NOT NULL DEFAULT '',    -- nama asli untuk Content-Disposition
  ext_url     TEXT NOT NULL DEFAULT '',    -- tautan eksternal (type=link)
  featured    INTEGER NOT NULL DEFAULT 0,
  active      INTEGER NOT NULL DEFAULT 1,
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE license_keys (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id    INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  key           TEXT NOT NULL,
  order_item_id INTEGER, -- NULL = masih stok
  created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE coupons (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  code       TEXT NOT NULL UNIQUE,
  percent    INTEGER NOT NULL, -- 1-90
  active     INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE orders (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  order_no    TEXT NOT NULL UNIQUE,      -- referensi publik (ORD-…)
  user_id     INTEGER,                   -- NULL = guest checkout
  email       TEXT NOT NULL,
  status      TEXT NOT NULL DEFAULT 'pending', -- pending | paid
  subtotal    INTEGER NOT NULL,
  discount    INTEGER NOT NULL DEFAULT 0,
  coupon_code TEXT NOT NULL DEFAULT '',
  total       INTEGER NOT NULL,
  pay_provider TEXT NOT NULL DEFAULT '',
  pay_url     TEXT NOT NULL DEFAULT '',
  pay_ref     TEXT NOT NULL DEFAULT '',
  dl_token    TEXT NOT NULL DEFAULT '',  -- dibuat saat checkout: kunci halaman bayar + unduhan
  reminded    INTEGER NOT NULL DEFAULT 0,
  created_at  TEXT NOT NULL DEFAULT (datetime('now')),
  paid_at     TEXT
);

CREATE TABLE order_items (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  order_id    INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  product_id  INTEGER NOT NULL REFERENCES products(id),
  title       TEXT NOT NULL,             -- snapshot judul saat checkout
  price       INTEGER NOT NULL,          -- snapshot harga saat checkout
  qty         INTEGER NOT NULL DEFAULT 1,
  license_key TEXT NOT NULL DEFAULT ''   -- diisi saat lunas (type=key)
);

CREATE TABLE tickets (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  order_no   TEXT NOT NULL DEFAULT '',
  subject    TEXT NOT NULL,
  status     TEXT NOT NULL DEFAULT 'open', -- open | answered | closed
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE ticket_messages (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  ticket_id  INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  sender     TEXT NOT NULL, -- user | admin
  body       TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE mail_log (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  to_email   TEXT NOT NULL,
  subject    TEXT NOT NULL,
  status     TEXT NOT NULL, -- sent | error
  error      TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE audit_log (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  action     TEXT NOT NULL,
  target     TEXT NOT NULL DEFAULT '',
  detail     TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_orders_email ON orders(email);
CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_license_keys_product ON license_keys(product_id, order_item_id);
CREATE INDEX idx_tickets_user ON tickets(user_id);
CREATE INDEX idx_ticket_messages_ticket ON ticket_messages(ticket_id);
