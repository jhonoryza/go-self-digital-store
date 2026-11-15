import { Head, Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import ProductThumb from "../../components/ProductThumb";
import { rupiah, type Product, type SharedProps } from "../../types";

interface Props extends SharedProps {
  products: Product[];
  seo?: { "seo.products.title"?: string; "seo.products.description"?: string };
}

export default function Products({ products, seo }: Props) {
  return (
    <>
      <Head>
        <title>{seo?.["seo.products.title"] || "Produk"}</title>
        {seo?.["seo.products.description"] && <meta name="description" content={seo["seo.products.description"]} />}
      </Head>
      <section className="mx-auto max-w-6xl px-4 py-12">
      <h1 className="mb-2 text-3xl font-bold">Semua Produk</h1>
      <p className="mb-8 text-zinc-600 dark:text-zinc-400">
        Produk digital siap unduh - terkirim otomatis ke email setelah lunas.
      </p>
      {products.length === 0 ? (
        <p className="rounded-xl border border-dashed border-zinc-300 p-12 text-center text-zinc-500 dark:border-zinc-700">
          Belum ada produk yang dijual. Segera hadir.
        </p>
      ) : (
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {products.map((p) => (
            <Link
              key={p.id}
              href={`/produk/${p.slug}`}
              className="group rounded-xl border border-zinc-200 bg-white p-5 transition hover:border-brand-400 hover:shadow-md dark:border-zinc-800 dark:bg-zinc-900"
            >
              <ProductThumb product={p} />
              <h3 className="font-semibold group-hover:text-brand-700 dark:group-hover:text-brand-300">{p.title}</h3>
              <p className="mt-1 line-clamp-2 text-sm text-zinc-600 dark:text-zinc-400">{p.description}</p>
              <div className="mt-3 flex items-center justify-between">
                <span className="font-bold text-brand-700 dark:text-brand-300">{rupiah(p.price)}</span>
                {p.featured && (
                  <span className="rounded-full bg-brass-400/20 px-2 py-0.5 text-xs font-semibold text-brass-600 dark:text-brass-400">
                    Unggulan
                  </span>
                )}
              </div>
            </Link>
          ))}
        </div>
      )}
    </section>
    </>
  );
}

Products.layout = (page: ReactNode) => <GuestLayout>{page}</GuestLayout>;
