import { Link, router } from "@inertiajs/react";
import { useState, type ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import { addToCart } from "../../lib/cart";
import { rupiah, type Product, type SharedProps } from "../../types";

interface Props extends SharedProps {
  product: Product;
}

export default function ProductDetail({ product }: Props) {
  const [added, setAdded] = useState(false);
  const add = () => {
    addToCart(product);
    setAdded(true);
    setTimeout(() => setAdded(false), 1500);
  };
  return (
    <section className="mx-auto max-w-4xl px-4 py-12">
      <Link href="/produk" className="mb-6 inline-block text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200">
        ← Semua produk
      </Link>
      <div className="rounded-xl border border-zinc-200 bg-white p-8 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="mb-4 flex h-32 items-center justify-center rounded-lg bg-brand-50 text-5xl dark:bg-brand-950">
          {product.type === "file" ? "📦" : product.type === "key" ? "🔑" : "🔗"}
        </div>
        <h1 className="text-3xl font-bold">{product.title}</h1>
        <p className="mt-4 whitespace-pre-wrap leading-relaxed text-zinc-600 dark:text-zinc-400">
          {product.description}
        </p>
        <div className="mt-8 flex flex-wrap items-center justify-between gap-4 border-t border-zinc-200 pt-6 dark:border-zinc-800">
          <span className="text-2xl font-extrabold text-brand-700 dark:text-brand-400">{rupiah(product.price)}</span>
          <div className="flex items-center gap-3">
            <button
              onClick={add}
              className="rounded-lg border border-brand-600 px-4 py-2.5 text-sm font-semibold text-brand-700 hover:bg-brand-50 dark:text-brand-400 dark:hover:bg-brand-950"
            >
              {added ? "✓ Ditambahkan" : "+ Keranjang"}
            </button>
            <button
              onClick={() => {
                addToCart(product);
                router.visit("/checkout");
              }}
              className="rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-700"
            >
              Beli Sekarang
            </button>
          </div>
        </div>
      </div>
    </section>
  );
}

ProductDetail.layout = (page: ReactNode) => <GuestLayout>{page}</GuestLayout>;
