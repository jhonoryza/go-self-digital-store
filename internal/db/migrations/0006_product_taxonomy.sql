-- Taksonomi produk opsional: kategori tunggal + tags (array JSON).
-- tags disimpan sebagai string JSON `["a","b"]` agar filter tag cukup LIKE
-- '%"tag"%' tanpa tabel pivot - token berkutip tidak mungkin tertukar substring.

ALTER TABLE products ADD COLUMN category TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN tags TEXT NOT NULL DEFAULT '[]';
