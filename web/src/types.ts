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
  image_url?: string; // gambar pertama (kartu katalog)
  meta_title?: string; // SEO opsional (form admin)
  meta_description?: string;
}

// gambar produk yang tersimpan di bucket R2 (maks 5 per produk)
export interface ProductImage {
  id: number;
  url: string;
}

// format rupiah gaya Indonesia (cermin helper Go rupiah).
export function rupiah(n: number): string {
  const neg = n < 0;
  const s = Math.abs(n).toLocaleString("id-ID"); // pemisah ribuan "."
  return (neg ? "-Rp" : "Rp") + s;
}
