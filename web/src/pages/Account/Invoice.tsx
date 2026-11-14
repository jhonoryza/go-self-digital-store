import { Download } from "lucide-react";
import { usePage } from "@inertiajs/react";
import type { ReactNode } from "react";
import AccountLayout from "../../components/AccountLayout";
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
  created_at: string;
  paid_at: string;
  items?: OrderItem[];
}

interface Props extends SharedProps {
  order: Order;
  token: string;
}

// Invoice + produk terkirim (dapat dicetak via Ctrl/Cmd+P).
export default function Invoice({ order, token }: Props) {
  const { app_name } = usePage<SharedProps>().props;
  const paid = order.status === "paid";
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between print:hidden">
        <h1 className="text-2xl font-bold">Invoice {order.order_no}</h1>
        <button
          onClick={() => window.print()}
          className="rounded-lg border border-zinc-300 px-3 py-1.5 text-sm hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
        >
          Cetak / PDF
        </button>
      </div>

      <div className="rounded-xl border border-zinc-200 bg-white p-8 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="mb-8 flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-lg font-bold">{app_name ?? "Toko Digital"}</p>
            <p className="text-sm text-zinc-500">Produk digital</p>
          </div>
          <div className="text-right text-sm">
            <p className="font-mono font-bold">{order.order_no}</p>
            <p className="text-zinc-500">Dibuat: {order.created_at}</p>
            {paid && <p className="text-zinc-500">Lunas: {order.paid_at}</p>}
            <span
              className={`mt-2 inline-block rounded-full px-2.5 py-0.5 text-xs font-semibold ${
                paid
                  ? "bg-brand-100 text-brand-800 dark:bg-brand-950 dark:text-brand-300"
                  : "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300"
              }`}
            >
              {paid ? "LUNAS" : "MENUNGGU PEMBAYARAN"}
            </span>
          </div>
        </div>

        <div className="mb-6 text-sm">
          <p className="text-zinc-500">Ditagihkan kepada</p>
          <p className="font-semibold">{order.email}</p>
        </div>

        <table className="w-full text-sm">
          <thead className="border-b border-zinc-200 text-left text-zinc-500 dark:border-zinc-800">
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
                <td className="py-2">{i.title}</td>
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
            <div className="flex justify-between text-brand-700 dark:text-brand-400">
              <span>Diskon {order.coupon_code}</span>
              <span>−{rupiah(order.discount)}</span>
            </div>
          )}
          <div className="flex justify-between border-t border-zinc-200 pt-1 text-base font-bold dark:border-zinc-800">
            <span>Total</span>
            <span>{rupiah(order.total)}</span>
          </div>
        </div>
      </div>

      {paid && (
        <div className="rounded-xl border border-brand-200 bg-brand-50 p-6 print:break-inside-avoid dark:border-brand-900 dark:bg-brand-950">
          <h2 className="mb-3 font-semibold">Produk Anda</h2>
          <div className="space-y-3">
            {(order.items ?? []).map((i) => (
              <div key={i.id} className="text-sm">
                <p className="font-semibold">{i.title}</p>
                {(i.license_keys ?? []).map((k) => (
                  <p key={k}>
                    Kunci: <span className="rounded bg-white px-2 py-0.5 font-mono font-bold dark:bg-zinc-900">{k}</span>
                  </p>
                ))}
                {i.type === "link" && i.ext_url && (
                  <a href={i.ext_url} className="text-brand-700 hover:underline dark:text-brand-400" target="_blank" rel="noreferrer">
                    Buka produk →
                  </a>
                )}
                {i.has_download && (
                  <a
                    href={`/dl/${order.order_no}/${i.id}?t=${token}`}
                    className="mt-1 inline-flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-brand-700"
                  >
                    <Download className="h-3.5 w-3.5" />
                    Unduh berkas
                  </a>
                )}
              </div>
            ))}
            <p className="text-xs text-zinc-500">Tautan unduh juga dikirim ke {order.email} saat pelunasan.</p>
          </div>
        </div>
      )}

      {!paid && (
        <p className="text-sm text-zinc-500 print:hidden">
          Order belum lunas -{" "}
          <a href={`/pay/${order.order_no}?t=${token}`} className="text-brand-600 hover:underline dark:text-brand-400">
            buka halaman pembayaran
          </a>
          .
        </p>
      )}
    </div>
  );
}

Invoice.layout = (page: ReactNode) => <AccountLayout>{page}</AccountLayout>;
