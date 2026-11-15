import { Head, Link } from "@inertiajs/react";
import { Compass } from "lucide-react";
import type { ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import type { SharedProps } from "../../types";

// Halaman 404: route tak dikenal atau produk/order yang sudah tidak ada.
export default function NotFound() {
  return (
    <>
      <Head>
        <title>404 - Halaman tidak ditemukan</title>
        <meta name="robots" content="noindex" />
      </Head>
      <section className="mx-auto flex max-w-xl flex-col items-center px-4 py-24 text-center">
        <Compass className="mb-6 h-14 w-14 text-brand-600 dark:text-brand-400" aria-hidden="true" />
        <p className="text-6xl font-extrabold tracking-tight">404</p>
        <h1 className="mt-3 text-xl font-bold">Halaman tidak ditemukan</h1>
        <p className="mt-2 text-zinc-600 dark:text-zinc-400">
          Tautan mungkin salah ketik, atau produk/order yang dicari sudah tidak tersedia.
        </p>
        <div className="mt-8 flex gap-3">
          <Link href="/" className="rounded-lg bg-brand-600 px-5 py-2.5 text-sm font-semibold text-white hover:bg-brand-700">
            Ke beranda
          </Link>
          <Link
            href="/produk"
            className="rounded-lg border border-zinc-300 px-5 py-2.5 text-sm font-semibold hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
          >
            Lihat produk
          </Link>
        </div>
      </section>
    </>
  );
}

NotFound.layout = (page: ReactNode) => <GuestLayout>{page}</GuestLayout>;
