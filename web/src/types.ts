// Tipe props bersama Inertia (dikirim oleh internal/web/helpers.go Page()).
import type { PageProps } from "@inertiajs/core";

export interface AuthUser {
  id: number;
  email: string;
  name: string;
  role: string;
}

export interface SharedProps extends PageProps {
  auth_user?: AuthUser;
  csrf_token?: string;
  flash?: { ok?: string; err?: string };
  app_name?: string;
  theme?: string; // tema toko pilihan admin ("" = bawaan; lihat app.css data-theme)
  brand_logo?: string; // URL logo toko dari bucket R2 (halaman setting admin)
  dev_login?: boolean; // pintu dev /dev/login aktif (STORE_DEV_LOGIN=1)
}

export interface DevAccount {
  id: number;
  email: string;
  name: string;
  role: string;
}

export interface Product {
  id: number;
  slug: string;
  title: string;
  description: string;
  price: number;
  type: string;
  featured: boolean;
  category?: string; // kategori opsional (taksonomi)
  tags?: string[]; // tag opsional (taksonomi)
  key_stock?: number; // sisa stok kunci (hanya type=key)
  image_url?: string; // gambar pertama (kartu katalog)
  meta_title?: string; // SEO opsional (form admin)
  meta_description?: string;
}

// state filter katalog yang dikirim balik server (cermiri catalogFilter Go)
export interface CatalogFilters {
  q?: string;
  kategori?: string;
  tag?: string;
  harga_min?: number;
  harga_max?: number;
  sort?: string;
}

// gambar produk yang tersimpan di bucket R2 (maks 5 per produk)
export interface ProductImage {
  id: number;
  url: string;
}

// AdminProduct baris produk di konsol admin (produk + kolom operasional).
export interface AdminProduct extends Product {
  active: boolean;
  file_path: string;
  file_name: string;
  ext_url: string;
  key_count: number;
}

// format rupiah gaya Indonesia (cermin helper Go rupiah).
export function rupiah(n: number): string {
  const neg = n < 0;
  const s = Math.abs(n).toLocaleString("id-ID"); // pemisah ribuan "."
  return (neg ? "-Rp" : "Rp") + s;
}
