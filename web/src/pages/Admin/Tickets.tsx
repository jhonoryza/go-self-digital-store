import { Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import type { SharedProps } from "../../types";

interface Props extends SharedProps {
  tickets: { id: number; order_no: string; subject: string; status: string; updated_at: string; email: string }[];
}

export default function AdminTickets({ tickets }: Props) {
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Tiket Bantuan</h1>
      <div className="space-y-3">
        {(tickets ?? []).map((t) => (
          <Link
            key={t.id}
            href={`/admin/tiket/${t.id}`}
            className="block rounded-xl border border-zinc-200 bg-white p-4 hover:border-brand-400 dark:border-zinc-800 dark:bg-zinc-900"
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-semibold">{t.subject}</span>
              <span
                className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${
                  t.status === "open"
                    ? "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300"
                    : t.status === "answered"
                      ? "bg-brand-100 text-brand-800 dark:bg-brand-950 dark:text-brand-300"
                      : "bg-zinc-200 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400"
                }`}
              >
                {t.status === "open" ? "Menunggu balasan" : t.status === "answered" ? "Sudah dibalas" : "Ditutup"}
              </span>
            </div>
            <p className="mt-1 text-xs text-zinc-500">
              {t.email}
              {t.order_no ? ` — order ${t.order_no}` : ""} — diperbarui {t.updated_at}
            </p>
          </Link>
        ))}
        {(tickets ?? []).length === 0 && (
          <p className="rounded-xl border border-dashed border-zinc-300 p-8 text-center text-zinc-500 dark:border-zinc-700">
            Tidak ada tiket.
          </p>
        )}
      </div>
    </div>
  );
}

AdminTickets.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
