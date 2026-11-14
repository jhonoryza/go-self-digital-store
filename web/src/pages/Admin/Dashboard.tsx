import { Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import { rupiah, type SharedProps } from "../../types";

interface Stats {
  revenue: number;
  orders_paid: number;
  orders_pending: number;
  users: number;
  tickets_open: number;
  mail_error: number;
  license_free: number;
  license_sold: number;
}

interface Props extends SharedProps {
  stats: Stats;
  recent: { order_no: string; email: string; status: string; total: number; created_at: string }[];
}

export default function AdminDashboard({ stats, recent }: Props) {
  const cards: [string, string, string][] = [
    ["Pendapatan lunas", rupiah(stats.revenue), "text-brand-700 dark:text-brand-400"],
    ["Order lunas", String(stats.orders_paid), ""],
    ["Order pending", String(stats.orders_pending), stats.orders_pending > 0 ? "text-amber-600" : ""],
    ["Pengguna", String(stats.users), ""],
    ["Tiket terbuka", String(stats.tickets_open), stats.tickets_open > 0 ? "text-amber-600" : ""],
    ["Email gagal", String(stats.mail_error), stats.mail_error > 0 ? "text-red-600" : ""],
    ["Stok kunci tersisa", String(stats.license_free), stats.license_free === 0 ? "text-red-600" : ""],
    ["Kunci terjual", String(stats.license_sold), ""],
  ];
  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-bold">Konsol Admin</h1>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {cards.map(([label, value, cls]) => (
          <div key={label} className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <p className="text-sm text-zinc-500">{label}</p>
            <p className={`mt-1 text-2xl font-extrabold ${cls}`}>{value}</p>
          </div>
        ))}
      </div>

      <div>
        <h2 className="mb-4 font-semibold">Order terbaru</h2>
        <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          <table className="w-full text-sm">
            <thead className="bg-zinc-50 text-left text-zinc-500 dark:bg-zinc-950">
              <tr>
                <th className="px-4 py-2.5 font-medium">Order</th>
                <th className="px-4 py-2.5 font-medium">Email</th>
                <th className="px-4 py-2.5 font-medium">Status</th>
                <th className="px-4 py-2.5 text-right font-medium">Total</th>
                <th className="px-4 py-2.5 font-medium">Waktu</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">
              {(recent ?? []).map((o) => (
                <tr key={o.order_no}>
                  <td className="px-4 py-2.5 font-mono">{o.order_no}</td>
                  <td className="px-4 py-2.5 text-zinc-600 dark:text-zinc-400">{o.email}</td>
                  <td className="px-4 py-2.5">{o.status === "paid" ? "Lunas" : "Pending"}</td>
                  <td className="px-4 py-2.5 text-right font-semibold">{rupiah(o.total)}</td>
                  <td className="px-4 py-2.5 text-zinc-500">{o.created_at}</td>
                </tr>
              ))}
              {(recent ?? []).length === 0 && (
                <tr><td colSpan={5} className="px-4 py-6 text-center text-zinc-500">Belum ada order.</td></tr>
              )}
            </tbody>
          </table>
        </div>
        <p className="mt-2 text-xs text-zinc-500">
          Tindakan lanjutan (tandai lunas, kirim ulang) di tab{" "}
          <Link href="/admin/orders" className="text-brand-600 hover:underline">Orders</Link>.
        </p>
      </div>
    </div>
  );
}

AdminDashboard.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
