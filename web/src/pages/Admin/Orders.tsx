import { router } from "@inertiajs/react";
import type { ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import { rupiah, type SharedProps } from "../../types";

interface Props extends SharedProps {
  orders: {
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
}

export default function AdminOrders({ orders }: Props) {
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Orders</h1>
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
            {(orders ?? []).map((o) => (
              <tr key={o.order_no}>
                <td className="px-4 py-2.5">
                  <span className="font-mono font-semibold">{o.order_no}</span>
                  <p className="text-xs text-zinc-500">{o.created_at}</p>
                </td>
                <td className="px-4 py-2.5 text-zinc-600 dark:text-zinc-400">{o.email}</td>
                <td className="px-4 py-2.5">{o.pay_provider || "—"}</td>
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
            {(orders ?? []).length === 0 && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-zinc-500">Belum ada order.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

AdminOrders.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
