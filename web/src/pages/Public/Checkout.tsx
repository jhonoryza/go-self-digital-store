import { Link, useForm, usePage } from "@inertiajs/react";
import { useEffect, useState, type ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import { cartTotal, getCart, clearCart, type CartItem } from "../../lib/cart";
import { rupiah, type SharedProps } from "../../types";

export default function Checkout({ auth_user }: SharedProps) {
  const { flash } = usePage<SharedProps>().props;
  const [items, setItems] = useState<CartItem[]>([]);
  useEffect(() => {
    setItems(getCart());
    const sync = () => setItems(getCart());
    window.addEventListener("pageshow", sync);
    return () => window.removeEventListener("pageshow", sync);
  }, []);

  const form = useForm({ items: [] as CartItem[], email: auth_user?.email ?? "", coupon: "" });

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    form.setData("items", items);
    form.post("/checkout", {
      onSuccess: () => {
        // order terbentuk - keranjang dikosongkan (redirect gateway/pay sudah jalan)
        if (items.length > 0 && !form.hasErrors) clearCart();
      },
    });
  };

  return (
    <section className="mx-auto max-w-4xl px-4 py-12">
      <h1 className="mb-8 text-2xl font-bold">Checkout</h1>
      {items.length === 0 ? (
        <div className="rounded-xl border border-dashed border-zinc-300 p-12 text-center dark:border-zinc-700">
          <p className="text-zinc-500">Keranjang kosong.</p>
          <Link href="/produk" className="mt-4 inline-block rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700">
            Jelajahi produk
          </Link>
        </div>
      ) : (
        <form onSubmit={submit} className="grid gap-6 md:grid-cols-5">
          <div className="space-y-4 md:col-span-3">
            <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
              <h2 className="mb-4 font-semibold">Email penerima produk</h2>
              <p className="mb-3 text-sm text-zinc-500">
                Produk digital & tautan unduh dikirim ke email ini setelah pembayaran terverifikasi.
              </p>
              <input
                id="email"
                type="email"
                className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
                placeholder="nama@email.com"
                value={form.data.email}
                onChange={(e) => form.setData("email", e.target.value)}
                required
              />
              {form.errors.email && <p className="mt-1 text-sm text-red-600">{form.errors.email}</p>}
              {!auth_user && (
                <p className="mt-3 text-xs text-zinc-500">
                  Sudah punya akun?{" "}
                  <Link href="/login" className="text-brand-600 hover:underline dark:text-brand-400">Masuk</Link>{" "}
                  agar pesanan tampil di dashboard. Checkout tanpa akun tetap bisa.
                </p>
              )}
            </div>
            <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
              <h2 className="mb-3 font-semibold">Kupon diskon (opsional)</h2>
              <input
                id="coupon"
                className="w-full rounded-lg border-zinc-300 uppercase dark:border-zinc-700 dark:bg-zinc-800"
                placeholder="KODEKUPON"
                value={form.data.coupon}
                onChange={(e) => form.setData("coupon", e.target.value)}
              />
            </div>
            {flash?.err && (
              <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-950 dark:text-red-400">{flash.err}</div>
            )}
          </div>

          <div className="rounded-xl border border-zinc-200 bg-white p-6 md:col-span-2 dark:border-zinc-800 dark:bg-zinc-900">
            <h2 className="mb-4 font-semibold">Ringkasan</h2>
            <ul className="mb-4 space-y-2 text-sm">
              {items.map((i) => (
                <li key={i.id} className="flex justify-between gap-2">
                  <span className="text-zinc-600 dark:text-zinc-400">
                    {i.title} ×{i.qty}
                  </span>
                  <span>{rupiah(i.price * i.qty)}</span>
                </li>
              ))}
            </ul>
            <div className="mb-6 flex justify-between border-t border-zinc-200 pt-4 text-lg font-bold dark:border-zinc-800">
              <span>Total</span>
              <span className="text-brand-700 dark:text-brand-300">{rupiah(cartTotal(items))}</span>
            </div>
            <button
              type="submit"
              disabled={form.processing}
              className="w-full rounded-lg bg-brand-600 px-4 py-3 font-semibold text-white hover:bg-brand-700 disabled:opacity-50"
            >
              {form.processing ? "Memproses…" : "Bayar sekarang"}
            </button>
            <p className="mt-3 text-xs text-zinc-500">
              Anda akan diarahkan ke halaman pembayaran gateway. Produk terkirim otomatis ke email setelah lunas.
            </p>
          </div>
        </form>
      )}
    </section>
  );
}

Checkout.layout = (page: ReactNode) => <GuestLayout>{page}</GuestLayout>;
