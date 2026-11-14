import { Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import { rupiah, type Product, type SharedProps } from "../../types";

interface Props extends SharedProps {
  product: Product;
}

export default function ProductDetail({ product }: Props) {
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
          <span className="rounded-full border border-zinc-300 px-3 py-1 text-xs font-medium text-zinc-600 dark:border-zinc-700 dark:text-zinc-400">
            {product.type === "file" ? "Berkas digital" : product.type === "key" ? "Kunci lisensi" : "Tautan eksternal"}
          </span>
        </div>
      </div>
    </section>
  );
}

ProductDetail.layout = (page: ReactNode) => <GuestLayout>{page}</GuestLayout>;
