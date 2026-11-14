import { Link } from "@inertiajs/react";
import { useEffect, useState, type ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import ProductThumb from "../../components/ProductThumb";
import { cartTotal, getCart, removeFromCart, setQty, type CartItem } from "../../lib/cart";
import { rupiah, type SharedProps } from "../../types";

export default function Cart({}: SharedProps) {
  const [items, setItems] = useState<CartItem[]>([]);
  useEffect(() => {
    setItems(getCart());
    const sync = () => setItems(getCart());
    window.addEventListener("cart-change", sync);
    window.addEventListener("pageshow", sync);
    return () => {
      window.removeEventListener("cart-change", sync);
      window.removeEventListener("pageshow", sync);
    };
  }, []);

  return (
    <section className="mx-auto max-w-4xl px-4 py-12">
      <h1 className="mb-8 text-2xl font-bold">Keranjang</h1>
      {items.length === 0 ? (
        <div className="rounded-xl border border-dashed border-zinc-300 p-12 text-center dark:border-zinc-700">
          <p className="text-zinc-500">Keranjang kosong.</p>
          <Link href="/produk" className="mt-4 inline-block rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700">
            Jelajahi produk
          </Link>
        </div>
      ) : (
        <div className="space-y-4">
          {items.map((i) => (
            <div key={i.id} className="flex items-center gap-4 rounded-xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
              <ProductThumb product={i} className="h-14 w-14 shrink-0" iconClassName="h-5 w-5" />
              <div className="min-w-0 flex-1">
                <Link href={`/produk/${i.slug}`} className="font-semibold hover:text-brand-700">{i.title}</Link>
                <p className="text-sm text-zinc-500">{rupiah(i.price)}</p>
              </div>
              <div className="flex items-center gap-1">
                <button
                  onClick={() => { setQty(i.id, i.qty - 1); setItems(getCart()); }}
                  className="h-8 w-8 rounded-md border border-zinc-300 hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
                  aria-label="Kurangi"
                >−</button>
                <span className="w-8 text-center font-semibold">{i.qty}</span>
                <button
                  onClick={() => { setQty(i.id, i.qty + 1); setItems(getCart()); }}
                  className="h-8 w-8 rounded-md border border-zinc-300 hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
                  aria-label="Tambah"
                >+</button>
              </div>
              <span className="w-28 text-right font-bold">{rupiah(i.price * i.qty)}</span>
              <button
                onClick={() => { removeFromCart(i.id); setItems(getCart()); }}
                className="text-zinc-400 hover:text-red-600"
                aria-label="Hapus"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2m3 0v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" />
                </svg>
              </button>
            </div>
          ))}
          <div className="flex items-center justify-between rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
            <span className="text-lg font-semibold">Total</span>
            <span className="text-2xl font-extrabold text-brand-700 dark:text-brand-400">{rupiah(cartTotal(items))}</span>
          </div>
          <Link
            href="/checkout"
            className="block rounded-xl bg-brand-600 px-6 py-3.5 text-center font-semibold text-white hover:bg-brand-700"
          >
            Lanjut ke pembayaran →
          </Link>
        </div>
      )}
    </section>
  );
}

Cart.layout = (page: ReactNode) => <GuestLayout>{page}</GuestLayout>;
