-- Login tanpa password: satu email memuat tautan ajaib (token) + kode OTP;
-- keduanya sekali pakai, TTL 15 menit, dan otomatis menandai email verified.

CREATE TABLE login_tokens (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  email      TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  code_hash  TEXT NOT NULL,
  attempts   INTEGER NOT NULL DEFAULT 0,
  expires_at TEXT NOT NULL,
  used_at    TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_login_tokens_email ON login_tokens(email);
