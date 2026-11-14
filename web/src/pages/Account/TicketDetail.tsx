import { Link, router, useForm } from "@inertiajs/react";
import { useEffect, type ReactNode } from "react";
import AccountLayout from "../../components/AccountLayout";
import type { SharedProps } from "../../types";

interface Message {
  id: number;
  sender: string; // user | admin
  body: string;
  created_at: string;
}

interface Ticket {
  id: number;
  order_no: string;
  subject: string;
  status: string;
  messages?: Message[];
}

interface Props extends SharedProps {
  ticket: Ticket;
}

// Percakapan tiket (non-realtime - polling ringan tiap 10 dtk hanya saat tab aktif).
export default function TicketDetail({ ticket }: Props) {
  const form = useForm({ body: "" });

  useEffect(() => {
    if (ticket.status === "closed") return;
    const t = setInterval(() => {
      if (document.visibilityState === "visible") {
        router.reload({ only: ["ticket"] });
      }
    }, 10_000);
    return () => clearInterval(t);
  }, [ticket.status]);

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <Link href="/tiket" className="inline-block text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200">
        ← Semua tiket
      </Link>
      <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h1 className="text-xl font-bold">{ticket.subject}</h1>
            {ticket.order_no && (
              <p className="text-sm text-zinc-500">
                Order <span className="font-mono">{ticket.order_no}</span>
              </p>
            )}
          </div>
          <span
            className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${
              ticket.status === "answered"
                ? "bg-brand-100 text-brand-800 dark:bg-brand-950 dark:text-brand-300"
                : ticket.status === "closed"
                  ? "bg-zinc-200 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400"
                  : "bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-300"
            }`}
          >
            {ticket.status === "answered" ? "Dibalas admin" : ticket.status === "closed" ? "Ditutup" : "Terbuka"}
          </span>
        </div>
      </div>

      <div className="space-y-3">
        {(ticket.messages ?? []).map((m) => (
          <div
            key={m.id}
            className={`max-w-[85%] rounded-xl px-4 py-3 text-sm ${
              m.sender === "admin"
                ? "border border-brand-200 bg-brand-50 dark:border-brand-900 dark:bg-brand-950"
                : "ml-auto border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
            }`}
          >
            <p className="mb-1 text-xs font-semibold text-zinc-500">
              {m.sender === "admin" ? "Admin" : "Anda"} - {m.created_at}
            </p>
            <p className="whitespace-pre-wrap">{m.body}</p>
          </div>
        ))}
      </div>

      {ticket.status !== "closed" ? (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            form.post(`/tiket/${ticket.id}`, { onSuccess: () => form.reset() });
          }}
          className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900"
        >
          <label htmlFor="body" className="mb-1 block text-sm font-medium">Balas</label>
          <textarea
            id="body"
            rows={3}
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.body}
            onChange={(e) => form.setData("body", e.target.value)}
            required
          />
          {form.errors.body && <p className="mt-1 text-sm text-red-600">{form.errors.body}</p>}
          <button
            type="submit"
            disabled={form.processing}
            className="mt-3 rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50"
          >
            {form.processing ? "Mengirim…" : "Kirim balasan"}
          </button>
        </form>
      ) : (
        <p className="rounded-xl border border-zinc-200 p-4 text-center text-sm text-zinc-500 dark:border-zinc-800">
          Tiket ditutup oleh admin.
        </p>
      )}
    </div>
  );
}

TicketDetail.layout = (page: ReactNode) => <AccountLayout>{page}</AccountLayout>;
