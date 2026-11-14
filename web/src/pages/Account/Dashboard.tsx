import { Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import AccountLayout from "../../components/AccountLayout";
import { rupiah, type SharedProps } from "../../types";

interface Order {
  order_no: string;
  status: string;
  total: number;
  created_at: string;
  items?: { title: string; qty: number }[];
}

interface Stats {
  paid_count: number;
  pending_count: number;
  total_spend: number;
  recent: Order[];
}

interface Props extends SharedProps {
  stats: Stats;
}

export default function Dashboard({ stats }: Props) {
  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
          <p className="text-sm text-zinc-500">Order lunas</p>
          <p className="mt-1 text-3xl font-extrabold">{stats.paid_count}</p>
        </div>
        <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
          <p className="text-sm text-zinc-500">Menunggu bayar</p>
          <p className="mt-1 text-3xl font-extrabold">{stats.pending_count}</p>
        </div>
        <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
          <p className="text-sm text-zinc-500">Total belanja</p>
          <p className="mt-1 text-3xl font-extrabold text-brand-700 dark:text-brand-400">{rupiah(stats.total_spend)}</p>
        </div>
      </div>

      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Pesanan terakhir</h2>
        <Link href="/pesanan" className="text-sm font-semibold text-brand-600 hover:underline dark:text-brand-400">
          Semua →
        </Link>
      </div>
      {(!stats.recent || stats.recent.length === 0) ? (
        <p className="rounded-xl border border-dashed border-zinc-300 p-8 text-center text-zinc-500 dark:border-zinc-700">
          Belum ada pesanan. <Link href="/produk" className="text-brand-600 hover:underline">Mulai belanja</Link>
        </p>
      ) : (
        <div className="space-y-3">
          {stats.recent.map((o) => (
            <Link
              key={o.order_no}
              href={`/pesanan/${o.order_no}`}
              className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-zinc-200 bg-white p-4 text-sm hover:border-brand-400 dark:border-zinc-800 dark:bg-zinc-900"
            >
              <div>
                <span className="font-mono font-semibold">{o.order_no}</span>
                <span className="ml-2 text-zinc-500">
                  {(o.items ?? []).map((i) => i.title).join(", ")}
                </span>
              </div>
              <div className="flex items-center gap-3">
                <span
                  className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${
                    o.status === "paid"
                      ? "bg-brand-100 text-brand-800 dark:bg-brand-950 dark:text-brand-300"
                      : "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300"
                  }`}
                >
                  {o.status === "paid" ? "Lunas" : "Menunggu"}
                </span>
                <span className="font-bold">{rupiah(o.total)}</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

Dashboard.layout = (page: ReactNode) => <AccountLayout>{page}</AccountLayout>;
