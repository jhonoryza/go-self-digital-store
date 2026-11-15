-- Temuan audit: OTP verifikasi email tanpa batas percobaan (brute force
-- 6 digit). Tambah counter attempts — mirror login_tokens (cap 5).
ALTER TABLE email_otps ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0;
