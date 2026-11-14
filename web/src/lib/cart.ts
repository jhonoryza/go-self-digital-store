// Keranjang belanja di localStorage — snapshot untuk tampilan; server
// memvalidasi ulang harga/ketersediaan saat checkout (internal/web/checkout.go).

import type { Product } from "../types";

export interface CartItem {
  id: number;
  slug: string;
  title: string;
  price: number;
  qty: number;
  type: string;
}

const KEY = "store_cart";

export function getCart(): CartItem[] {
  try {
    const raw = localStorage.getItem(KEY);
    const items = raw ? JSON.parse(raw) : [];
    return Array.isArray(items) ? items : [];
  } catch {
    return [];
  }
}

function save(items: CartItem[]) {
  localStorage.setItem(KEY, JSON.stringify(items));
  window.dispatchEvent(new Event("cart-change"));
}

export function addToCart(p: Product) {
  const items = getCart();
  const found = items.find((i) => i.id === p.id);
  if (found) {
    found.qty = Math.min(found.qty + 1, 10);
  } else {
    items.push({ id: p.id, slug: p.slug, title: p.title, price: p.price, qty: 1, type: p.type });
  }
  save(items);
}

export function setQty(id: number, qty: number) {
  const items = getCart().filter((i) => i.id !== id);
  const item = getCart().find((i) => i.id === id);
  if (item && qty > 0) items.push({ ...item, qty: Math.min(qty, 10) });
  save(items);
}

export function removeFromCart(id: number) {
  save(getCart().filter((i) => i.id !== id));
}

export function clearCart() {
  save([]);
}

export function cartTotal(items: CartItem[]): number {
  return items.reduce((sum, i) => sum + i.price * i.qty, 0);
}

export function cartCount(items: CartItem[]): number {
  return items.reduce((sum, i) => sum + i.qty, 0);
}
