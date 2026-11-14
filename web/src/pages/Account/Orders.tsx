import { Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import AccountLayout from "../../components/AccountLayout";
import { rupiah, type SharedProps } from "../../types";

interface Order {
  order_no: string;
  status: string;
  total: number;
  paid_at: string;
  created_at: string;
  items?: { title: string; qty: number }[];
}

interface Props extends SharedProps {
  orders: Order[];
}

export default function Orders({ orders }: Props) {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Pesanan Saya</h1>
      {(!orders || orders.length === 0) ? (
        <p className="rounded-xl border border-dashed border-zinc-300 p-12 text-center text-zinc-500 dark:border-zinc-700">
          Belum ada pesanan. <Link href="/produk" className="text-brand-600 hover:underline">Mulai belanja →</Link>
        </p>
      ) : (
        <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
          <table className="w-full text-sm">
            <thead className="bg-zinc-50 text-left text-zinc-500 dark:bg-zinc-950">
              <tr>
                <th className="px-4 py-3 font-medium">Order</th>
                <th className="px-4 py-3 font-medium">Produk</th>
                <th className="px-4 py-3 font-medium">Total</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">
              {orders.map((o) => (
                <tr key={o.order_no}>
                  <td className="px-4 py-3">
                    <span className="font-mono font-semibold">{o.order_no}</span>
                    <p className="text-xs text-zinc-500">{o.created_at}</p>
                  </td>
                  <td className="px-4 py-3 text-zinc-600 dark:text-zinc-400">
                    {(o.items ?? []).map((i) => `${i.title} ×${i.qty}`).join(", ") || "—"}
                  </td>
                  <td className="px-4 py-3 font-semibold">{rupiah(o.total)}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${
                        o.status === "paid"
                          ? "bg-brand-100 text-brand-800 dark:bg-brand-950 dark:text-brand-300"
                          : "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300"
                      }`}
                    >
                      {o.status === "paid" ? "Lunas" : "Menunggu"}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Link href={`/pesanan/${o.order_no}`} className="text-brand-600 hover:underline dark:text-brand-400">
                      Invoice →
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

Orders.layout = (page: ReactNode) => <AccountLayout>{page}</AccountLayout>;
