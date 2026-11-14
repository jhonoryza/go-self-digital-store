import { router, usePage } from "@inertiajs/react";
import { LogIn, ShieldCheck, UserRound } from "lucide-react";
import { useState } from "react";
import GuestLayout from "../../components/GuestLayout";
import { type DevAccount, type SharedProps } from "../../types";

// Dev login-as: pilih akun mana pun untuk masuk tanpa password.
// Hanya tersedia saat server dijalankan dengan STORE_DEV_LOGIN=1.
export default function DevLoginAs() {
  const { accounts } = usePage<SharedProps>().props as unknown as { accounts: DevAccount[] };
  const [q, setQ] = useState("");
  const needle = q.trim().toLowerCase();
  const match = (a: DevAccount) =>
    !needle || a.email.toLowerCase().includes(needle) || a.name.toLowerCase().includes(needle);
  const admins = accounts.filter((a) => a.role === "admin" && match(a));
  const users = accounts.filter((a) => a.role !== "admin" && match(a));

  const loginAs = (a: DevAccount) => {
    router.post("/dev/login", { user_id: a.id });
  };

  return (
    <GuestLayout>
      <section className="mx-auto max-w-3xl px-4 py-12">
        <p className="mb-1 text-xs font-semibold uppercase tracking-widest text-brass-600 dark:text-brass-400">
          Alat development
        </p>
        <h1 className="text-2xl font-bold tracking-tight">Masuk sebagai akun</h1>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          Pembuat sesi langsung tanpa password. Aktif hanya saat server berjalan dengan
          <code className="mx-1 rounded bg-zinc-100 px-1.5 py-0.5 font-mono text-xs dark:bg-zinc-800">STORE_DEV_LOGIN=1</code>
          - route ini tidak terdaftar di produksi.
        </p>

        <input
          type="search"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Cari email atau nama…"
          aria-label="Cari akun"
          className="mt-6 w-full rounded-lg border border-zinc-300 bg-white px-3.5 py-2.5 text-sm outline-none focus:border-brand-500 dark:border-zinc-700 dark:bg-zinc-900"
        />

        <AccountGroup icon={<ShieldCheck className="h-4 w-4" />} label="Admin" accounts={admins} onPick={loginAs} />
        <AccountGroup icon={<UserRound className="h-4 w-4" />} label="User" accounts={users} onPick={loginAs} />
      </section>
    </GuestLayout>
  );
}

function AccountGroup({
  icon,
  label,
  accounts,
  onPick,
}: {
  icon: React.ReactNode;
  label: string;
  accounts: DevAccount[];
  onPick: (a: DevAccount) => void;
}) {
  if (accounts.length === 0) return null;
  return (
    <div className="mt-8">
      <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-zinc-500">
        {icon}
        {label} ({accounts.length})
      </h2>
      <ul className="divide-y divide-zinc-200 overflow-hidden rounded-xl border border-zinc-200 bg-white dark:divide-zinc-800 dark:border-zinc-800 dark:bg-zinc-900">
        {accounts.map((a) => (
          <li key={a.id} className="flex items-center justify-between gap-3 px-4 py-3">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{a.name || "-"}</p>
              <p className="truncate text-xs text-zinc-500">{a.email}</p>
            </div>
            <button
              onClick={() => onPick(a)}
              className="inline-flex shrink-0 items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-brand-700"
            >
              <LogIn className="h-3.5 w-3.5" />
              Masuk
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
