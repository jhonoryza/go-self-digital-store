import { Link } from "@inertiajs/react";
import { RotateCcw, Search } from "lucide-react";
import type { ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import FilterPanel from "../../components/FilterPanel";
import { useInfiniteList } from "../../lib/infinite";
import type { SharedProps } from "../../types";

interface Filters {
  q?: string;
  status?: string;
  sort?: string;
}

interface Props extends SharedProps {
  tickets?: { id: number; order_no: string; subject: string; status: string; updated_at: string; email: string }[];
  page: number;
  last_page: number;
  total: number;
  filters?: Filters;
}

const SORTS: Record<string, string> = {
  terbaru: "Terbaru diperbarui",
  terlama: "Terlama diperbarui",
  subjek: "Subjek A-Z",
};

// Konsol tiket: pencarian subjek/email/no. order, filter status, urutan,
// dan infinite scroll.
export default function AdminTickets({ tickets: rawTickets, page, last_page, total, filters = {} }: Props) {
  const tickets = rawTickets ?? [];
  const nFilter = [filters.q, filters.status].filter(Boolean).length;
  const { items, busy, sentinelRef } = useInfiniteList(
    tickets,
    page,
    last_page,
    JSON.stringify(filters),
    ["tickets", "page", "last_page", "total"],
  );

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Tiket Bantuan</h1>

      <FilterPanel activeCount={nFilter} id="filter-tiket">
        <form method="get" action="/admin/tiket" className="rounded-xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <div className="lg:col-span-2">
              <label htmlFor="q" className="mb-1 block text-xs font-medium text-zinc-500">
                Cari subjek / email / no. order
              </label>
              <input
                id="q"
                name="q"
                type="search"
                defaultValue={filters.q || ""}
                placeholder="Cari tiket…"
                className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
              />
            </div>
            <div>
              <label htmlFor="status" className="mb-1 block text-xs font-medium text-zinc-500">
                Status
              </label>
              <select id="status" name="status" defaultValue={filters.status || ""} className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800">
                <option value="">Semua</option>
                <option value="open">Menunggu balasan</option>
                <option value="answered">Sudah dibalas</option>
                <option value="closed">Ditutup</option>
              </select>
            </div>
            <div>
              <label htmlFor="sort" className="mb-1 block text-xs font-medium text-zinc-500">
                Urutkan
              </label>
              <select id="sort" name="sort" defaultValue={filters.sort || "terbaru"} className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800">
                {Object.entries(SORTS).map(([v, label]) => (
                  <option key={v} value={v}>
                    {label}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div className="mt-3 flex items-center gap-3">
            <button type="submit" className="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700">
              <Search className="h-4 w-4" aria-hidden="true" /> Terapkan
            </button>
            {nFilter > 0 && (
              <a href="/admin/tiket" className="inline-flex items-center gap-1.5 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200">
                <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" /> Reset
              </a>
            )}
            <span className="ml-auto text-sm text-zinc-500">{total} tiket</span>
          </div>
        </form>
      </FilterPanel>

      <div className="space-y-3">
        {items.map((t) => (
          <Link
            key={t.id}
            href={`/admin/tiket/${t.id}`}
            className="block rounded-xl border border-zinc-200 bg-white p-4 hover:border-brand-400 dark:border-zinc-800 dark:bg-zinc-900"
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-semibold">{t.subject}</span>
              <span
                className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${
                  t.status === "open"
                    ? "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300"
                    : t.status === "answered"
                      ? "bg-brand-100 text-brand-800 dark:bg-brand-950 dark:text-brand-300"
                      : "bg-zinc-200 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400"
                }`}
              >
                {t.status === "open" ? "Menunggu balasan" : t.status === "answered" ? "Sudah dibalas" : "Ditutup"}
              </span>
            </div>
            <p className="mt-1 text-xs text-zinc-500">
              {t.email}
              {t.order_no ? ` - order ${t.order_no}` : ""} - diperbarui {t.updated_at}
            </p>
          </Link>
        ))}
        {items.length === 0 && (
          <p className="rounded-xl border border-dashed border-zinc-300 p-8 text-center text-zinc-500 dark:border-zinc-700">
            {nFilter > 0 ? "Tidak ada tiket yang cocok dengan filter." : "Tidak ada tiket."}
          </p>
        )}
        <div ref={sentinelRef} aria-hidden="true" className="h-1" />
        {busy &&
          [0, 1].map((i) => (
            <div key={`sk-${i}`} aria-hidden="true" className="animate-pulse rounded-xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
              <div className="h-4 w-1/3 rounded bg-zinc-200 dark:bg-zinc-800" />
              <div className="mt-2 h-3 w-1/2 rounded bg-zinc-100 dark:bg-zinc-800/60" />
            </div>
          ))}
      </div>
    </div>
  );
}

AdminTickets.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
