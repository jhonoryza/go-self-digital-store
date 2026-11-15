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
  pay_provider: string;
  created_at: string;
  paid_at: string;
  items?: OrderItem[];
}

interface Props extends SharedProps {
  order: Order;
  token: string;
}

// label metode pembayaran yang ramah pembaca (provider tersimpan sebagai nama
// pendek di settings, mis. mock/tripay).
function providerLabel(name: string): string {
  if (!name) return "Simulasi (QRIS mock)";
  const map: Record<string, string> = { mock: "Simulasi (QRIS mock)" };
  return map[name] ?? name.charAt(0).toUpperCase() + name.slice(1);
}

// Invoice + produk terkirim. Tampilan layar mengikuti tema; hasil cetak
// dipaksa tinta gelap di atas kertas lewat @media print di app.css.
export default function Invoice({ order, token }: Props) {
  const { app_name } = usePage<SharedProps>().props;
  const paid = order.status === "paid";
  const brand = app_name ?? "Toko Digital";
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
        {/* Kepala faktur: penjual kiri, judul + status kanan */}
        <div className="flex flex-wrap items-start justify-between gap-4 border-b border-zinc-200 pb-6 dark:border-zinc-800 print:border-zinc-400">
          <div>
            <p className="text-xl font-bold tracking-tight">{brand}</p>
            <p className="text-sm text-zinc-500">Produk digital</p>
            <p className="mt-1 text-xs text-zinc-500">Faktur dibuat otomatis oleh sistem</p>
          </div>
          <div className="text-right">
            <p className="text-2xl font-extrabold uppercase tracking-tight">Invoice</p>
            <span
              className={`mt-1 inline-block rounded-full px-2.5 py-0.5 text-xs font-semibold print:border print:border-zinc-400 ${
                paid
                  ? "bg-brand-100 text-brand-800 dark:bg-brand-950 dark:text-brand-300"
                  : "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300"
              }`}
            >
              {paid ? "LUNAS" : "MENUNGGU PEMBAYARAN"}
            </span>
          </div>
        </div>

        {/* Info faktur: pembeli kiri, tanggal & metode kanan */}
        <div className="grid gap-6 border-b border-zinc-200 py-6 text-sm sm:grid-cols-2 dark:border-zinc-800 print:border-zinc-400">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-zinc-500">Ditagihkan kepada</p>
            <p className="mt-1 font-semibold">{order.email}</p>
            <p className="text-zinc-500">Produk dikirim otomatis ke email ini.</p>
          </div>
          <div className="sm:text-right">
            <p className="text-xs font-semibold uppercase tracking-wide text-zinc-500">Detail faktur</p>
            <p className="mt-1">
              No. <span className="font-mono font-semibold">{order.order_no}</span>
            </p>
            <p className="text-zinc-500">Terbit: {order.created_at}</p>
            {paid && order.paid_at && <p className="text-zinc-500">Lunas: {order.paid_at}</p>}
            <p className="text-zinc-500">Metode: {providerLabel(order.pay_provider)}</p>
          </div>
        </div>

        {/* Rincian item */}
        <table className="mt-6 w-full text-sm">
          <thead className="border-b border-zinc-200 text-left text-xs uppercase tracking-wide text-zinc-500 dark:border-zinc-800 print:border-zinc-400">
            <tr>
              <th className="py-2 font-medium">Produk</th>
              <th className="py-2 text-center font-medium">Qty</th>
              <th className="py-2 text-right font-medium">Harga</th>
              <th className="py-2 text-right font-medium">Subtotal</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800 print:divide-zinc-300">
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
            <div className="flex justify-between">
              <span className="text-zinc-500">Diskon {order.coupon_code}</span>
              <span>−{rupiah(order.discount)}</span>
            </div>
          )}
          <div className="flex justify-between border-t border-zinc-200 pt-1 text-base font-bold dark:border-zinc-800 print:border-zinc-400">
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
                  <a href={i.ext_url} className="text-brand-700 hover:underline dark:text-brand-300" target="_blank" rel="noreferrer">
                    Buka produk →
                  </a>
                )}
                {i.has_download && (
                  <a
                    href={`/dl/${order.order_no}/${i.id}?t=${token}`}
                    className="mt-1 inline-flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-brand-700 print:hidden"
                  >
                    <Download className="h-3.5 w-3.5" />
                    Unduh berkas
                  </a>
                )}
              </div>
            ))}
            <p className="text-xs text-zinc-500 print:hidden">
              Tautan unduh juga dikirim ke {order.email} saat pelunasan.
            </p>
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

      <p className="text-center text-xs text-zinc-500">
        Terima kasih telah berbelanja di {brand}. Faktur ini sah tanpa tanda tangan basah.
      </p>
    </div>
  );
}

Invoice.layout = (page: ReactNode) => <AccountLayout>{page}</AccountLayout>;
