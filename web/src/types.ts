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
}

export interface Product {
  id: number;
  slug: string;
  title: string;
  description: string;
  price: number;
  type: string;
  featured: boolean;
}

// format rupiah gaya Indonesia (cermin helper Go rupiah).
export function rupiah(n: number): string {
  const neg = n < 0;
  let s = Math.abs(n).toString();
  const start = s.length % 3;
  if (start > 0 && s.length > 3) s = s.slice(0, start) + "." + s.slice(start);
  for (let i = start + 3; i < s.length; i += 4) {
    s = s.slice(0, i) + "." + s.slice(i);
  }
  return (neg ? "-Rp" : "Rp") + s;
}
