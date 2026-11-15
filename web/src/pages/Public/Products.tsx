import { Head, Link } from "@inertiajs/react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import type { ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import ProductThumb from "../../components/ProductThumb";
import { rupiah, type Product, type SharedProps } from "../../types";

interface Props extends SharedProps {
  products: Product[];
  page: number;
  last_page: number;
  total: number;
  seo?: { "seo.products.title"?: string; "seo.products.description"?: string };
}

// Nomor halaman di sekitar halaman aktif (maks 5) dengan halaman 1 dan terakhir selalu tampil.
function pageItems(current: number, last: number): (number | "…")[] {
  const around = [current - 1, current, current + 1].filter((n) => n >= 2 && n <= last - 1);
  const uniq = [...new Set(around)].sort((a, b) => a - b);
  const out: (number | "…")[] = [1];
  if (uniq.length && uniq[0] > 2) out.push("…");
  out.push(...uniq);
  if (uniq.length && uniq[uniq.length - 1] < last - 1) out.push("…");
  if (last > 1) out.push(last);
  return out;
}

export default function Products({ products, page, last_page, total, seo }: Props) {
  const from = total === 0 ? 0 : (page - 1) * products.length + 1;
  const to = (page - 1) * products.length + products.length;

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
          <>
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

            {last_page > 1 && (
              <nav aria-label="Navigasi halaman" className="mt-10 flex flex-col items-center gap-3">
                <p className="text-sm text-zinc-500">
                  Menampilkan {from} sampai {to} dari {total} produk
                </p>
                <div className="flex items-center gap-1">
                  {page > 1 ? (
                    <Link
                      href={`/produk?page=${page - 1}`}
                      aria-label="Halaman sebelumnya"
                      className="flex h-9 w-9 items-center justify-center rounded-lg border border-zinc-200 text-zinc-600 hover:border-brand-400 hover:text-brand-600 dark:border-zinc-800 dark:text-zinc-400 dark:hover:text-brand-400"
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </Link>
                  ) : (
                    <span
                      aria-disabled="true"
                      className="flex h-9 w-9 items-center justify-center rounded-lg border border-zinc-100 text-zinc-300 dark:border-zinc-900 dark:text-zinc-700"
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </span>
                  )}
                  {pageItems(page, last_page).map((n, i) =>
                    n === "…" ? (
                      <span key={`gap-${i}`} className="px-1.5 text-zinc-400">
                        …
                      </span>
                    ) : n === page ? (
                      <span
                        key={n}
                        aria-current="page"
                        className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-600 text-sm font-semibold text-white"
                      >
                        {n}
                      </span>
                    ) : (
                      <Link
                        key={n}
                        href={`/produk?page=${n}`}
                        className="flex h-9 w-9 items-center justify-center rounded-lg border border-zinc-200 text-sm text-zinc-600 hover:border-brand-400 hover:text-brand-600 dark:border-zinc-800 dark:text-zinc-400 dark:hover:text-brand-400"
                      >
                        {n}
                      </Link>
                    ),
                  )}
                  {page < last_page ? (
                    <Link
                      href={`/produk?page=${page + 1}`}
                      aria-label="Halaman berikutnya"
                      className="flex h-9 w-9 items-center justify-center rounded-lg border border-zinc-200 text-zinc-600 hover:border-brand-400 hover:text-brand-600 dark:border-zinc-800 dark:text-zinc-400 dark:hover:text-brand-400"
                    >
                      <ChevronRight className="h-4 w-4" />
                    </Link>
                  ) : (
                    <span
                      aria-disabled="true"
                      className="flex h-9 w-9 items-center justify-center rounded-lg border border-zinc-100 text-zinc-300 dark:border-zinc-900 dark:text-zinc-700"
                    >
                      <ChevronRight className="h-4 w-4" />
                    </span>
                  )}
                </div>
              </nav>
            )}
          </>
        )}
      </section>
    </>
  );
}

Products.layout = (page: ReactNode) => <GuestLayout>{page}</GuestLayout>;
