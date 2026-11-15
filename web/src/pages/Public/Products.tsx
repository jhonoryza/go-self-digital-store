import { Head, Link } from "@inertiajs/react";
import { ChevronDown, ChevronLeft, ChevronRight, RotateCcw, Search, SlidersHorizontal } from "lucide-react";
import { useState, type ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import ProductThumb from "../../components/ProductThumb";
import { rupiah, type CatalogFilters, type Product, type SharedProps } from "../../types";

interface Props extends SharedProps {
  products: Product[];
  page: number;
  last_page: number;
  total: number;
  categories: string[]; // kategori unik utk dropdown filter
  tags: string[]; // tag unik utk dropdown filter
  filters: CatalogFilters; // state filter aktif (echo dari query string)
  seo?: { "seo.products.title"?: string; "seo.products.description"?: string };
}

const SORT_LABELS: Record<string, string> = {
  terbaru: "Terbaru",
  termurah: "Harga termurah",
  termahal: "Harga termahal",
  terlama: "Terlama",
};

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

// katalogURL rakit /produk dengan filter aktif + halaman (paginasi mempertahankan filter).
function katalogURL(f: CatalogFilters, page: number): string {
  const p = new URLSearchParams();
  if (f.q) p.set("q", f.q);
  if (f.kategori) p.set("kategori", f.kategori);
  if (f.tag) p.set("tag", f.tag);
  if (f.harga_min) p.set("harga_min", String(f.harga_min));
  if (f.harga_max) p.set("harga_max", String(f.harga_max));
  if (f.sort && f.sort !== "terbaru") p.set("sort", f.sort);
  if (page > 1) p.set("page", String(page));
  const s = p.toString();
  return s ? `/produk?${s}` : "/produk";
}

export default function Products({
  products: rawProducts,
  page,
  last_page,
  total,
  categories: rawCategories,
  tags: rawTags,
  filters = {},
  seo,
}: Props) {
  // props nil (katalog kosong / tanpa kategori-tag) dihilangkan gonertia
  const products = rawProducts ?? [];
  const categories = rawCategories ?? [];
  const tags = rawTags ?? [];
  const from = total === 0 ? 0 : (page - 1) * products.length + 1;
  const to = (page - 1) * products.length + products.length;
  const adaFilter = Boolean(filters.q || filters.kategori || filters.tag || filters.harga_min || filters.harga_max);
  const filterCount = [filters.q, filters.kategori, filters.tag, filters.harga_min, filters.harga_max].filter(Boolean).length;
  // panel tersembunyi secara default; terbuka otomatis saat ada filter aktif
  const [filterOpen, setFilterOpen] = useState(adaFilter);


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

        {/* Tombol buka/tutup panel filter; angka = jumlah filter aktif */}
        <div className="mb-4">
          <button
            type="button"
            onClick={() => setFilterOpen(!filterOpen)}
            aria-expanded={filterOpen}
            aria-controls="panel-filter"
            className="inline-flex items-center gap-2 rounded-lg border border-zinc-300 bg-white px-4 py-2 text-sm font-semibold text-zinc-700 hover:border-brand-400 hover:text-brand-700 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:border-brand-500 dark:hover:text-brand-300"
          >
            <SlidersHorizontal className="h-4 w-4" aria-hidden="true" />
            Filter
            {filterCount > 0 && (
              <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-brand-600 px-1.5 text-[10px] font-bold text-white">
                {filterCount}
              </span>
            )}
            <ChevronDown className={`h-4 w-4 text-zinc-400 transition-transform ${filterOpen ? "rotate-180" : ""}`} aria-hidden="true" />
          </button>
        </div>

        {/* Panel cari + filter + urut: satu form GET, server yang menyaring */}
        {filterOpen && (
        <form
          id="panel-filter"
          method="get"
          action="/produk"
          className="mb-8 rounded-xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900"
        >
          <p className="mb-3 flex items-center gap-2 text-sm font-semibold">
            <SlidersHorizontal className="h-4 w-4 text-brand-600 dark:text-brand-400" aria-hidden="true" />
            Cari &amp; filter
          </p>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <div className="sm:col-span-2 lg:col-span-1">
              <label htmlFor="q" className="mb-1 block text-xs font-medium text-zinc-500">
                Nama produk
              </label>
              <input
                id="q"
                name="q"
                type="search"
                defaultValue={filters.q || ""}
                placeholder="Cari nama produk…"
                className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
              />
            </div>
            <div>
              <label htmlFor="kategori" className="mb-1 block text-xs font-medium text-zinc-500">
                Kategori
              </label>
              <select
                id="kategori"
                name="kategori"
                defaultValue={filters.kategori || ""}
                className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
              >
                <option value="">Semua kategori</option>
                {categories.map((c) => (
                  <option key={c} value={c}>
                    {c}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label htmlFor="tag" className="mb-1 block text-xs font-medium text-zinc-500">
                Tag
              </label>
              <select
                id="tag"
                name="tag"
                defaultValue={filters.tag || ""}
                className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
              >
                <option value="">Semua tag</option>
                {tags.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label htmlFor="harga_min" className="mb-1 block text-xs font-medium text-zinc-500">
                Harga minimum
              </label>
              <input
                id="harga_min"
                name="harga_min"
                type="number"
                min="0"
                defaultValue={filters.harga_min || ""}
                placeholder="0"
                className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
              />
            </div>
            <div>
              <label htmlFor="harga_max" className="mb-1 block text-xs font-medium text-zinc-500">
                Harga maksimum
              </label>
              <input
                id="harga_max"
                name="harga_max"
                type="number"
                min="0"
                defaultValue={filters.harga_max || ""}
                placeholder="Tak terbatas"
                className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
              />
            </div>
            <div>
              <label htmlFor="sort" className="mb-1 block text-xs font-medium text-zinc-500">
                Urutkan
              </label>
              <select
                id="sort"
                name="sort"
                defaultValue={filters.sort || "terbaru"}
                className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
              >
                {Object.entries(SORT_LABELS).map(([v, label]) => (
                  <option key={v} value={v}>
                    {label}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div className="mt-3 flex items-center gap-3">
            <button
              type="submit"
              className="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"
            >
              <Search className="h-4 w-4" aria-hidden="true" />
              Terapkan
            </button>
            {adaFilter && (
              <Link
                href="/produk"
                className="inline-flex items-center gap-1.5 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
              >
                <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
                Reset
              </Link>
            )}
          </div>
        </form>
        )}

        {products.length === 0 ? (
          <p className="rounded-xl border border-dashed border-zinc-300 p-12 text-center text-zinc-500 dark:border-zinc-700">
            {adaFilter
              ? "Tidak ada produk yang cocok dengan pencarian atau filter."
              : "Belum ada produk yang dijual. Segera hadir."}
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
                  {p.category && (
                    <p className="mt-2 text-xs font-semibold uppercase tracking-wide text-brand-600 dark:text-brand-400">
                      {p.category}
                    </p>
                  )}
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
                      href={katalogURL(filters, page - 1)}
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
                        href={katalogURL(filters, n)}
                        className="flex h-9 w-9 items-center justify-center rounded-lg border border-zinc-200 text-sm text-zinc-600 hover:border-brand-400 hover:text-brand-600 dark:border-zinc-800 dark:text-zinc-400 dark:hover:text-brand-400"
                      >
                        {n}
                      </Link>
                    ),
                  )}
                  {page < last_page ? (
                    <Link
                      href={katalogURL(filters, page + 1)}
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
