import { router, usePage } from "@inertiajs/react";
import { useEffect, type ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import { clearCart } from "../../lib/cart";
import { rupiah, type SharedProps } from "../../types";
import { CheckCircle2, Hourglass } from "lucide-react";

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
  pay_url: string;
  pay_ref: string;
  pay_provider: string;
  paid_at: string;
  items?: OrderItem[];
}

interface Props extends SharedProps {
  order: Order;
  token: string;
  simulasi: boolean;
  provider: string;
}

// Halaman bayar: status order, CTA gateway / simulasi, dan (bila lunas)
// produk terkirim (kunci lisensi / tautan / unduhan).
export default function Pay({ order, token, simulasi }: Props) {
  const paid = order.status === "paid";

  // Kunjungan /pay selalu pasca-checkout - keranjang sudah diproses.
  useEffect(() => {
    clearCart();
  }, []);

  const simulate = () => router.post(`/pay/${order.order_no}`, { t: token });

  return (
    <section className="mx-auto max-w-2xl px-4 py-12">
      <div
        className={`mb-6 flex items-center gap-3 rounded-xl px-6 py-4 ${
          paid ? "bg-brand-50 text-brand-800 dark:bg-brand-950 dark:text-brand-300" : "bg-amber-50 text-amber-800 dark:bg-amber-950 dark:text-amber-300"
        }`}
      >
        {paid ? <CheckCircle2 className="h-10 w-10 text-brand-600 dark:text-brand-400" /> : <Hourglass className="h-10 w-10 text-brass-500" aria-hidden="true" />}
        <div>
          <p className="font-bold">{paid ? "Pembayaran terverifikasi!" : "Menunggu pembayaran"}</p>
          <p className="text-sm">
            Order <span className="font-mono font-semibold">{order.order_no}</span>
            {paid && order.paid_at ? ` - lunas ${order.paid_at}` : ""}
          </p>
        </div>
      </div>

      <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <h2 className="mb-4 font-semibold">Rincian order</h2>
        <ul className="space-y-2 text-sm">
          {(order.items ?? []).map((i) => (
            <li key={i.id} className="flex justify-between">
              <span className="text-zinc-600 dark:text-zinc-400">{i.title} ×{i.qty}</span>
              <span>{rupiah(i.price * i.qty)}</span>
            </li>
          ))}
          {order.discount > 0 && (
            <li className="flex justify-between text-brand-700 dark:text-brand-400">
              <span>Diskon {order.coupon_code}</span>
              <span>−{rupiah(order.discount)}</span>
            </li>
          )}
        </ul>
        <div className="mt-4 flex justify-between border-t border-zinc-200 pt-4 text-lg font-bold dark:border-zinc-800">
          <span>Total</span>
          <span className="text-brand-700 dark:text-brand-400">{rupiah(order.total)}</span>
        </div>
      </div>

      {!paid && (
        <div className="mt-6 space-y-4">
          {order.pay_url ? (
            <a
              href={order.pay_url}
              className="block rounded-xl bg-brand-600 px-6 py-4 text-center text-lg font-semibold text-white hover:bg-brand-700"
            >
              Bayar sekarang →
            </a>
          ) : (
            simulasi && (
              <button
                onClick={simulate}
                className="w-full rounded-xl bg-amber-500 px-6 py-4 text-center text-lg font-semibold text-white hover:bg-amber-600"
              >
                Tandai lunas (mode simulasi)
              </button>
            )
          )}
          {order.pay_ref && (
            <div className="rounded-xl border border-zinc-200 bg-white p-4 text-center text-sm dark:border-zinc-800 dark:bg-zinc-900">
              <span className="text-zinc-500">Kode pembayaran: </span>
              <span className="font-mono font-bold">{order.pay_ref}</span>
            </div>
          )}
          <p className="text-center text-sm text-zinc-500">
            Produk otomatis terkirim ke <span className="font-semibold">{order.email}</span> begitu
            pembayaran terverifikasi. Simpan halaman ini (tautan) untuk cek status.
          </p>
        </div>
      )}

      {paid && (
        <div className="mt-6 space-y-4">
          {(order.items ?? []).map((i) => (
            <div key={i.id} className="rounded-xl border border-brand-200 bg-brand-50 p-4 dark:border-brand-900 dark:bg-brand-950">
              <p className="font-semibold">{i.title}</p>
              {(i.license_keys ?? []).map((k) => (
                <p key={k} className="mt-1 text-sm">
                  Kunci lisensi: <span className="rounded bg-white px-2 py-1 font-mono font-bold dark:bg-zinc-900">{k}</span>
                </p>
              ))}
              {i.ext_url && (
                <a href={i.ext_url} className="mt-1 inline-block text-sm font-semibold text-brand-700 hover:underline dark:text-brand-400">
                  Buka produk →
                </a>
              )}
              {i.has_download && (
                <a href={`/dl/${order.order_no}/${i.id}?t=${token}`} className="mt-1 inline-block text-sm font-semibold text-brand-700 hover:underline dark:text-brand-400">
                  Unduh berkas →
                </a>
              )}
            </div>
          ))}
          <p className="text-center text-sm text-zinc-500">
            Semua detail juga dikirim ke <span className="font-semibold">{order.email}</span>.
          </p>
        </div>
      )}
    </section>
  );
}

Pay.layout = (page: ReactNode) => <GuestLayout>{page}</GuestLayout>;
