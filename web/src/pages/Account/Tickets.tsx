import { Link, useForm } from "@inertiajs/react";
import type { ReactNode } from "react";
import AccountLayout from "../../components/AccountLayout";
import type { SharedProps } from "../../types";

interface Ticket {
  id: number;
  order_no: string;
  subject: string;
  status: string;
  updated_at: string;
}

interface Props extends SharedProps {
  tickets: Ticket[];
}

const statusLabel: Record<string, [string, string]> = {
  open: ["Terbuka", "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300"],
  answered: ["Dibalas admin", "bg-brand-100 text-brand-800 dark:bg-brand-950 dark:text-brand-300"],
  closed: ["Ditutup", "bg-zinc-200 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400"],
};

export default function Tickets({ tickets }: Props) {
  const form = useForm({ subject: "", order_no: "", body: "" });
  return (
    <div className="grid gap-8 lg:grid-cols-2">
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">Tiket Bantuan</h1>
        {(!tickets || tickets.length === 0) ? (
          <p className="rounded-xl border border-dashed border-zinc-300 p-8 text-center text-sm text-zinc-500 dark:border-zinc-700">
            Belum ada tiket. Buat tiket untuk masalah order/pembayaran.
          </p>
        ) : (
          <div className="space-y-3">
            {tickets.map((t) => {
              const [label, cls] = statusLabel[t.status] ?? [t.status, ""];
              return (
                <Link
                  key={t.id}
                  href={`/tiket/${t.id}`}
                  className="block rounded-xl border border-zinc-200 bg-white p-4 hover:border-brand-400 dark:border-zinc-800 dark:bg-zinc-900"
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-semibold">{t.subject}</span>
                    <span className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${cls}`}>{label}</span>
                  </div>
                  <p className="mt-1 text-xs text-zinc-500">
                    {t.order_no ? `Order ${t.order_no} — ` : ""}diperbarui {t.updated_at}
                  </p>
                </Link>
              );
            })}
          </div>
        )}
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          form.post("/tiket", { onSuccess: () => form.reset() });
        }}
        className="space-y-4 rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900"
      >
        <h2 className="font-semibold">Buat tiket baru</h2>
        <div>
          <label htmlFor="subject" className="mb-1 block text-sm font-medium">Subjek</label>
          <input
            id="subject"
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.subject}
            onChange={(e) => form.setData("subject", e.target.value)}
            required
          />
          {form.errors.subject && <p className="mt-1 text-sm text-red-600">{form.errors.subject}</p>}
        </div>
        <div>
          <label htmlFor="order_no" className="mb-1 block text-sm font-medium">
            Order terkait (opsional) — <span className="font-mono">ORD-…</span>
          </label>
          <input
            id="order_no"
            className="w-full rounded-lg border-zinc-300 font-mono uppercase dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.order_no}
            onChange={(e) => form.setData("order_no", e.target.value)}
            placeholder="ORD-XXXXXXXX"
          />
          {form.errors.order_no && <p className="mt-1 text-sm text-red-600">{form.errors.order_no}</p>}
        </div>
        <div>
          <label htmlFor="body" className="mb-1 block text-sm font-medium">Jelaskan masalah</label>
          <textarea
            id="body"
            rows={5}
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.body}
            onChange={(e) => form.setData("body", e.target.value)}
            required
          />
          {form.errors.body && <p className="mt-1 text-sm text-red-600">{form.errors.body}</p>}
        </div>
        <button
          type="submit"
          disabled={form.processing}
          className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50"
        >
          {form.processing ? "Mengirim…" : "Kirim tiket"}
        </button>
      </form>
    </div>
  );
}

Tickets.layout = (page: ReactNode) => <AccountLayout>{page}</AccountLayout>;
