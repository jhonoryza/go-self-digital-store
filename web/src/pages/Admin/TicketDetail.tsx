import { Link, router, useForm } from "@inertiajs/react";
import type { ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import type { SharedProps } from "../../types";

interface Message {
  id: number;
  sender: string;
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
  email: string;
}

export default function AdminTicketDetail({ ticket, email }: Props) {
  const form = useForm({ body: "" });
  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <Link href="/admin/tiket" className="inline-block text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200">
        ← Semua tiket
      </Link>
      <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <h1 className="text-xl font-bold">{ticket.subject}</h1>
        <p className="mt-1 text-sm text-zinc-500">
          {email}
          {ticket.order_no ? ` — order ${ticket.order_no}` : ""} — status: {ticket.status}
        </p>
      </div>

      <div className="space-y-3">
        {(ticket.messages ?? []).map((m) => (
          <div
            key={m.id}
            className={`max-w-[85%] rounded-xl px-4 py-3 text-sm ${
              m.sender === "admin"
                ? "ml-auto border border-brand-200 bg-brand-50 dark:border-brand-900 dark:bg-brand-950"
                : "border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
            }`}
          >
            <p className="mb-1 text-xs font-semibold text-zinc-500">
              {m.sender === "admin" ? "Admin (Anda)" : "User"} — {m.created_at}
            </p>
            <p className="whitespace-pre-wrap">{m.body}</p>
          </div>
        ))}
      </div>

      {ticket.status !== "closed" ? (
        <div className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
          <form
            onSubmit={(e) => {
              e.preventDefault();
              form.post(`/admin/tiket/${ticket.id}`, { onSuccess: () => form.reset() });
            }}
          >
            <label htmlFor="body" className="mb-1 block text-sm font-medium">Balas sebagai admin</label>
            <textarea
              id="body"
              rows={3}
              className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
              value={form.data.body}
              onChange={(e) => form.setData("body", e.target.value)}
              required
            />
            {form.errors.body && <p className="mt-1 text-sm text-red-600">{form.errors.body}</p>}
            <div className="mt-3 flex gap-2">
              <button type="submit" disabled={form.processing}
                className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50">
                {form.processing ? "Mengirim…" : "Kirim balasan"}
              </button>
              <button
                type="button"
                onClick={() => router.post(`/admin/tiket/${ticket.id}/close`)}
                className="rounded-lg border border-zinc-300 px-4 py-2 text-sm font-semibold hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
              >
                Tutup tiket
              </button>
            </div>
          </form>
        </div>
      ) : (
        <p className="rounded-xl border border-zinc-200 p-4 text-center text-sm text-zinc-500 dark:border-zinc-800">
          Tiket sudah ditutup.
        </p>
      )}
    </div>
  );
}

AdminTicketDetail.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
