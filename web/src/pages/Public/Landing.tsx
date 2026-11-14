import { Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import { rupiah, type Product, type SharedProps } from "../../types";

interface Props extends SharedProps {
  featured: Product[];
}

export default function Landing({ featured }: Props) {
  return (
    <>
      {/* Hero */}
      <section className="border-b border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div className="mx-auto max-w-6xl px-4 py-20 text-center sm:py-28">
          <p className="mb-3 text-sm font-semibold uppercase tracking-widest text-brand-600 dark:text-brand-400">
            Produk digital, langsung terkirim
          </p>
          <h1 className="mx-auto max-w-3xl text-4xl font-extrabold tracking-tight sm:text-5xl">
            Beli sekali, unduh segera setelah pembayaran terverifikasi
          </h1>
          <p className="mx-auto mt-5 max-w-2xl text-lg text-zinc-600 dark:text-zinc-400">
            Template, e-book, tools, dan aset digital lainnya. Bayar lewat gateway favoritmu —
            tautan unduh otomatis masuk ke email.
          </p>
          <div className="mt-8 flex items-center justify-center gap-3">
            <Link
              href="/produk"
              className="rounded-lg bg-brand-600 px-5 py-3 text-sm font-semibold text-white hover:bg-brand-700"
            >
              Lihat Produk
            </Link>
            <Link
              href="/ketentuan"
              className="rounded-lg border border-zinc-300 px-5 py-3 text-sm font-semibold hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
            >
              Cara Kerja
            </Link>
          </div>
        </div>
      </section>

      {/* Keunggulan */}
      <section className="mx-auto max-w-6xl px-4 py-16">
        <div className="grid gap-6 sm:grid-cols-3">
          {[
            ["Pengiriman otomatis", "Setelah pembayaran terverifikasi, produk terkirim ke email dalam hitungan detik."],
            ["Pembayaran beragam", "QRIS, VA bank, e-wallet, hingga checkout internasional — via gateway terpercaya."],
            ["Tautan tahan lama", "Tautan unduh tetap hidup; ulangi unduhan kapan pun dari halaman bayar."],
          ].map(([title, body]) => (
            <div key={title} className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
              <h3 className="font-semibold">{title}</h3>
              <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">{body}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Produk unggulan */}
      {featured.length > 0 && (
        <section className="mx-auto max-w-6xl px-4 pb-20">
          <div className="mb-6 flex items-center justify-between">
            <h2 className="text-2xl font-bold">Produk Unggulan</h2>
            <Link href="/produk" className="text-sm font-semibold text-brand-600 hover:text-brand-800 dark:text-brand-400">
              Semua produk →
            </Link>
          </div>
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {featured.map((p) => (
              <Link
                key={p.id}
                href={`/produk/${p.slug}`}
                className="group rounded-xl border border-zinc-200 bg-white p-5 transition hover:border-brand-400 hover:shadow-md dark:border-zinc-800 dark:bg-zinc-900"
              >
                <div className="mb-3 flex h-28 items-center justify-center rounded-lg bg-brand-50 text-3xl dark:bg-brand-950">
                  {p.type === "file" ? "📦" : p.type === "key" ? "🔑" : "🔗"}
                </div>
                <h3 className="font-semibold group-hover:text-brand-700 dark:group-hover:text-brand-400">{p.title}</h3>
                <p className="mt-1 line-clamp-2 text-sm text-zinc-600 dark:text-zinc-400">{p.description}</p>
                <p className="mt-3 font-bold text-brand-700 dark:text-brand-400">{rupiah(p.price)}</p>
              </Link>
            ))}
          </div>
        </section>
      )}

      {featured.length === 0 && (
        <section className="mx-auto max-w-6xl px-4 pb-20 text-center text-zinc-500">
          Belum ada produk — segera hadir.
        </section>
      )}
    </>
  );
}

Landing.layout = (page: ReactNode) => <GuestLayout>{page}</GuestLayout>;
