-- Verifikasi email: pengguna lama dianggap sudah terverifikasi (tidak
-- mengunci akun existing); pengguna baru diverifikasi lewat OTP email.

ALTER TABLE users ADD COLUMN email_verified_at TEXT;

UPDATE users SET email_verified_at = datetime('now') WHERE email_verified_at IS NULL;

CREATE TABLE email_otps (
  user_id      INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  code         TEXT NOT NULL,
  expires_at   TEXT NOT NULL,
  last_sent_at TEXT NOT NULL DEFAULT (datetime('now'))
);
