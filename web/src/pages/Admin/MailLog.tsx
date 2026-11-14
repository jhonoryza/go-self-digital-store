import type { ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import type { SharedProps } from "../../types";

interface Props extends SharedProps {
  logs: { to: string; subject: string; status: string; error: string; created_at: string }[];
}

export default function AdminMailLog({ logs }: Props) {
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Mail Log</h1>
      <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <table className="w-full text-sm">
          <thead className="bg-zinc-50 text-left text-zinc-500 dark:bg-zinc-950">
            <tr>
              <th className="px-4 py-2.5 font-medium">Kepada</th>
              <th className="px-4 py-2.5 font-medium">Subjek</th>
              <th className="px-4 py-2.5 font-medium">Status</th>
              <th className="px-4 py-2.5 font-medium">Error</th>
              <th className="px-4 py-2.5 font-medium">Waktu</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">
            {(logs ?? []).map((l, i) => (
              <tr key={i}>
                <td className="px-4 py-2.5">{l.to}</td>
                <td className="px-4 py-2.5 text-zinc-600 dark:text-zinc-400">{l.subject}</td>
                <td className="px-4 py-2.5">
                  <span
                    className={`rounded-full px-2 py-0.5 text-xs font-semibold ${
                      l.status === "sent"
                        ? "bg-brand-100 text-brand-800 dark:bg-brand-950 dark:text-brand-300"
                        : "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400"
                    }`}
                  >
                    {l.status}
                  </span>
                </td>
                <td className="max-w-xs truncate px-4 py-2.5 text-xs text-zinc-500" title={l.error}>{l.error || "-"}</td>
                <td className="px-4 py-2.5 text-zinc-500">{l.created_at}</td>
              </tr>
            ))}
            {(logs ?? []).length === 0 && (
              <tr><td colSpan={5} className="px-4 py-6 text-center text-zinc-500">Belum ada email terkirim.</td></tr>
            )}
          </tbody>
        </table>
      </div>
      <p className="text-xs text-zinc-500">
        Email dikirim via SMTP (Brevo). Saat SMTP belum dikonfigurasi, upaya tercatat sebagai error.
      </p>
    </div>
  );
}

AdminMailLog.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
