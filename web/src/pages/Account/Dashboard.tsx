import { Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import AccountLayout from "../../components/AccountLayout";
import type { SharedProps } from "../../types";

export default function Dashboard({}: SharedProps) {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      <div className="grid gap-4 sm:grid-cols-3">
        <Link href="/pesanan" className="rounded-xl border border-zinc-200 bg-white p-6 hover:border-brand-400 dark:border-zinc-800 dark:bg-zinc-900">
          <p className="text-sm text-zinc-500">Pesanan saya</p>
          <p className="mt-2 text-lg font-semibold">Lihat riwayat →</p>
        </Link>
        <Link href="/tiket" className="rounded-xl border border-zinc-200 bg-white p-6 hover:border-brand-400 dark:border-zinc-800 dark:bg-zinc-900">
          <p className="text-sm text-zinc-500">Ada masalah order?</p>
          <p className="mt-2 text-lg font-semibold">Buka tiket →</p>
        </Link>
        <Link href="/produk" className="rounded-xl border border-zinc-200 bg-white p-6 hover:border-brand-400 dark:border-zinc-800 dark:bg-zinc-900">
          <p className="text-sm text-zinc-500">Jelajahi katalog</p>
          <p className="mt-2 text-lg font-semibold">Produk →</p>
        </Link>
      </div>
      <p className="text-sm text-zinc-500">Ringkasan pembelian & unduhan menyusul di milestone berikutnya.</p>
    </div>
  );
}

Dashboard.layout = (page: ReactNode) => <AccountLayout>{page}</AccountLayout>;
