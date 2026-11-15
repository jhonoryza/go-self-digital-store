import { Link, router } from "@inertiajs/react";
import { Loader2, RotateCcw, Search } from "lucide-react";
import type { ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import FilterPanel from "../../components/FilterPanel";
import { useInfiniteList } from "../../lib/infinite";
import { rupiah, type SharedProps } from "../../types";

interface Filters {
  q?: string;
  status?: string;
  sort?: string;
}

interface Props extends SharedProps {
  orders?: {
    order_no: string;
    email: string;
    status: string;
    subtotal: number;
    discount: number;
    total: number;
    pay_provider: string;
    paid_at: string;
    created_at: string;
  }[];
  page: number;
  last_page: number;
  total: number;
  filters?: Filters;
}

const SORTS: Record<string, string> = {
  terbaru: "Terbaru",
  terlama: "Terlama",
  termahal: "Total termahal",
  termurah: "Total termurah",
};

// Konsol order: pencarian no/email, filter status, urutan, infinite scroll.
export default function AdminOrders({ orders: rawOrders, page, last_page, total, filters = {} }: Props) {
  const orders = rawOrders ?? [];
  const nFilter = [filters.q, filters.status].filter(Boolean).length;
  const { items, busy, sentinelRef } = useInfiniteList(
    orders,
    page,
    last_page,
    JSON.stringify(filters),
    ["orders", "page", "last_page", "total"],
  );

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Orders</h1>

      <FilterPanel activeCount={nFilter} id="filter-orders">
        <form method="get" action="/admin/orders" className="rounded-xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <div className="lg:col-span-2">
              <label htmlFor="q" className="mb-1 block text-xs font-medium text-zinc-500">
                Cari no. order / email
              </label>
              <input
                id="q"
                name="q"
                type="search"
                defaultValue={filters.q || ""}
                placeholder="ORD-… atau email pembeli"
                className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
              />
            </div>
            <div>
              <label htmlFor="status" className="mb-1 block text-xs font-medium text-zinc-500">
                Status
              </label>
              <select id="status" name="status" defaultValue={filters.status || ""} className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800">
                <option value="">Semua</option>
                <option value="paid">Lunas</option>
                <option value="pending">Pending</option>
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
              <a href="/admin/orders" className="inline-flex items-center gap-1.5 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200">
                <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" /> Reset
              </a>
            )}
            <span className="ml-auto text-sm text-zinc-500">{total} order</span>
          </div>
        </form>
      </FilterPanel>

      <div className="overflow-x-auto rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <table className="w-full text-sm">
          <thead className="bg-zinc-50 text-left text-zinc-500 dark:bg-zinc-950">
            <tr>
              <th className="px-4 py-2.5 font-medium">Order</th>
              <th className="px-4 py-2.5 font-medium">Email</th>
              <th className="px-4 py-2.5 font-medium">Gateway</th>
              <th className="px-4 py-2.5 font-medium">Total</th>
              <th className="px-4 py-2.5 font-medium">Status</th>
              <th className="px-4 py-2.5 font-medium">Aksi</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">
            {items.map((o) => (
              <tr key={o.order_no}>
                <td className="px-4 py-2.5">
                  <Link href={`/admin/orders/${o.order_no}`} className="font-mono font-semibold text-brand-700 hover:underline dark:text-brand-400">
                    {o.order_no}
                  </Link>
                  <p className="text-xs text-zinc-500">{o.created_at}</p>
                </td>
                <td className="px-4 py-2.5 text-zinc-600 dark:text-zinc-400">{o.email}</td>
                <td className="px-4 py-2.5">{o.pay_provider || "-"}</td>
                <td className="px-4 py-2.5 font-semibold">{rupiah(o.total)}</td>
                <td className="px-4 py-2.5">
                  <span
                    className={`rounded-full px-2 py-0.5 text-xs font-semibold ${
                      o.status === "paid"
                        ? "bg-brand-100 text-brand-800 dark:bg-brand-950 dark:text-brand-300"
                        : "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300"
                    }`}
                  >
                    {o.status === "paid" ? `Lunas ${o.paid_at}` : "Pending"}
                  </span>
                </td>
                <td className="px-4 py-2.5">
                  <div className="flex gap-2">
                    {o.status !== "paid" && (
                      <button
                        onClick={() => router.post(`/admin/orders/${o.order_no}/paid`)}
                        className="rounded-md bg-brand-600 px-2.5 py-1 text-xs font-semibold text-white hover:bg-brand-700"
                      >
                        Tandai lunas
                      </button>
                    )}
                    {o.status === "paid" && (
                      <button
                        onClick={() => router.post(`/admin/orders/${o.order_no}/resend`)}
                        className="rounded-md border border-zinc-300 px-2.5 py-1 text-xs font-semibold hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
                      >
                        Kirim ulang email
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
            {items.length === 0 && (
              <tr>
                <td colSpan={6} className="px-4 py-6 text-center text-zinc-500">
                  {nFilter > 0 ? "Tidak ada order yang cocok dengan filter." : "Belum ada order."}
                </td>
              </tr>
            )}
          </tbody>
        </table>
        <div ref={sentinelRef} aria-hidden="true" className="h-1" />
        {busy && (
          <p className="flex items-center justify-center gap-2 py-3 text-sm text-zinc-500">
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> Memuat order lain…
          </p>
        )}
      </div>
    </div>
  );
}

AdminOrders.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
