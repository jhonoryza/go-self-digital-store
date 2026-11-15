import { Link, router } from "@inertiajs/react";
import { Download, Mail } from "lucide-react";
import type { ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import { rupiah, type SharedProps } from "../../types";

interface OrderItem {
  id: number;
  title: string;
  price: number;
  qty: number;
  type: string;
  license_keys?: string[];
  ext_url?: string;
  has_download: boolean;
}

interface Order {
  order_no: string;
  email: string;
  status: string;
  subtotal: number;
  discount: number;
  coupon_code: string;
  total: number;
  pay_provider: string;
  pay_ref?: string;
  created_at: string;
  paid_at: string;
  items?: OrderItem[];
}

interface Props extends SharedProps {
  order: Order;
  token: string;
}

// Detail order versi admin: sama lengkapnya dengan invoice user + aksi admin.
export default function AdminOrderDetail({ order, token }: Props) {
  const paid = order.status === "paid";
  return (
    <div className="space-y-6">
      <Link href="/admin/orders" className="inline-block text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200">
        ← Semua order
      </Link>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold">
          Order <span className="font-mono">{order.order_no}</span>
        </h1>
        <span
          className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${
            paid
              ? "bg-brand-100 text-brand-800 dark:bg-brand-950 dark:text-brand-300"
              : "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300"
          }`}
        >
          {paid ? "LUNAS" : "MENUNGGU PEMBAYARAN"}
        </span>
      </div>

      {/* Info pembeli + pembayaran */}
      <div className="grid gap-4 rounded-xl border border-zinc-200 bg-white p-6 text-sm sm:grid-cols-3 dark:border-zinc-800 dark:bg-zinc-900">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-zinc-500">Pembeli</p>
          <p className="mt-1 font-semibold">{order.email}</p>
        </div>
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-zinc-500">Pembayaran</p>
          <p className="mt-1">
            {order.pay_provider || "simulasi"}
            {order.pay_ref && <span className="ml-1 font-mono text-zinc-500">({order.pay_ref})</span>}
          </p>
        </div>
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-zinc-500">Waktu</p>
          <p className="mt-1">Dibuat: {order.created_at}</p>
          {paid && order.paid_at && <p className="text-zinc-500">Lunas: {order.paid_at}</p>}
        </div>
      </div>

      {/* Rincian */}
      <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <table className="w-full text-sm">
          <thead className="border-b border-zinc-200 text-left text-xs uppercase tracking-wide text-zinc-500 dark:border-zinc-800">
            <tr>
              <th className="py-2 font-medium">Produk</th>
              <th className="py-2 text-center font-medium">Qty</th>
              <th className="py-2 text-right font-medium">Harga</th>
              <th className="py-2 text-right font-medium">Subtotal</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">
            {(order.items ?? []).map((i) => (
              <tr key={i.id}>
                <td className="py-2">
                  {i.title}
                  {i.type === "key" && paid && (i.license_keys?.length ?? 0) < i.qty && (
                    <p className="mt-0.5 text-xs font-semibold text-amber-600">
                      Stok kunci kurang: terpasang {i.license_keys?.length ?? 0} dari {i.qty} - isi pool kunci lalu kirim ulang email.
                    </p>
                  )}
                </td>
                <td className="py-2 text-center">{i.qty}</td>
                <td className="py-2 text-right">{rupiah(i.price)}</td>
                <td className="py-2 text-right">{rupiah(i.price * i.qty)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className="mt-4 ml-auto max-w-xs space-y-1 text-sm">
          <div className="flex justify-between">
            <span className="text-zinc-500">Subtotal</span>
            <span>{rupiah(order.subtotal)}</span>
          </div>
          {order.discount > 0 && (
            <div className="flex justify-between">
              <span className="text-zinc-500">Diskon {order.coupon_code}</span>
              <span>−{rupiah(order.discount)}</span>
            </div>
          )}
          <div className="flex justify-between border-t border-zinc-200 pt-1 text-base font-bold dark:border-zinc-800">
            <span>Total</span>
            <span>{rupiah(order.total)}</span>
          </div>
        </div>
      </div>

      {/* Produk terkirim */}
      {paid && (
        <div className="rounded-xl border border-brand-200 bg-brand-50 p-6 dark:border-brand-900 dark:bg-brand-950">
          <h2 className="mb-3 font-semibold">Produk terkirim</h2>
          <div className="space-y-3 text-sm">
            {(order.items ?? []).map((i) => (
              <div key={i.id}>
                <p className="font-semibold">{i.title}</p>
                {(i.license_keys ?? []).map((k) => (
                  <p key={k}>
                    Kunci: <span className="rounded bg-white px-2 py-0.5 font-mono font-bold dark:bg-zinc-900">{k}</span>
                  </p>
                ))}
                {i.type === "link" && i.ext_url && (
                  <a href={i.ext_url} target="_blank" rel="noreferrer" className="text-brand-700 hover:underline dark:text-brand-400">
                    {i.ext_url}
                  </a>
                )}
                {i.has_download && (
                  <a
                    href={`/dl/${order.order_no}/${i.id}?t=${token}`}
                    className="mt-1 inline-flex items-center gap-1.5 rounded-lg bg-zinc-900 px-3 py-1.5 text-xs font-semibold text-white hover:bg-zinc-800 dark:bg-white dark:text-zinc-900"
                  >
                    <Download className="h-3.5 w-3.5" />
                    Unduh (uji tautan pembeli)
                  </a>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Aksi admin */}
      <div className="flex flex-wrap gap-2">
        {!paid && (
          <button
            onClick={() => router.post(`/admin/orders/${order.order_no}/paid`)}
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"
          >
            Tandai lunas + kirim produk
          </button>
        )}
        {paid && (
          <button
            onClick={() => router.post(`/admin/orders/${order.order_no}/resend`)}
            className="inline-flex items-center gap-1.5 rounded-lg border border-zinc-300 px-4 py-2 text-sm font-semibold hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
          >
            <Mail className="h-4 w-4" />
            Kirim ulang email produk
          </button>
        )}
      </div>
    </div>
  );
}

AdminOrderDetail.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
