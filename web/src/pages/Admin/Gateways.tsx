import { router, usePage } from "@inertiajs/react";
import { useState, type ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import type { SharedProps } from "../../types";

interface Field {
  key: string;
  label: string;
  secret: boolean;
  value: string;
}

interface Provider {
  name: string;
  label: string;
  simulated: boolean;
  fields: Field[];
}

interface Props extends SharedProps {
  active: string;
  providers: Provider[];
}

// Konfigurasi gateway paygateway: field dinamis dari p.Fields(), secret masked.
export default function AdminGateways({ active, providers }: Props) {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Gateway Pembayaran</h1>
      <p className="text-sm text-zinc-500">
        Aktif saat ini: <span className="font-mono font-bold text-zinc-800 dark:text-zinc-200">{active}</span>.
        Nilai tersimpan tampil tersamar - biarkan kosong bila tidak ingin mengubah.
      </p>
      <div className="grid gap-6 lg:grid-cols-2">
        {(providers ?? []).map((p) => (
          <GatewayCard key={p.name} provider={p} active={active} />
        ))}
      </div>
    </div>
  );
}

function GatewayCard({ provider, active }: { provider: Provider; active: string }) {
  const [values, setValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(provider.fields.map((f) => [f.key, f.value === "•••" ? "" : f.value])),
  );
  const [saving, setSaving] = useState(false);
  const isActive = active === provider.name;

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    router.post(
      "/admin/gateway/config",
      { provider: provider.name, ...values },
      { onFinish: () => setSaving(false) },
    );
  };

  return (
    <form
      onSubmit={submit}
      className={`rounded-xl border-2 bg-white p-6 dark:bg-zinc-900 ${
        isActive ? "border-brand-500" : "border-zinc-200 dark:border-zinc-800"
      }`}
    >
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h2 className="font-bold">{provider.label}</h2>
          <p className="font-mono text-xs text-zinc-500">{provider.name}</p>
        </div>
        {provider.simulated ? (
          <span className="rounded-full bg-zinc-200 px-2.5 py-0.5 text-xs font-semibold text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400">
            simulasi
          </span>
        ) : isActive ? (
          <span className="rounded-full bg-brand-100 px-2.5 py-0.5 text-xs font-semibold text-brand-800 dark:bg-brand-950 dark:text-brand-300">
            AKTIF
          </span>
        ) : null}
      </div>
      <div className="space-y-3">
        {provider.fields.map((f) => (
          <div key={f.key}>
            <label htmlFor={f.key} className="mb-1 block text-xs font-medium">{f.label}</label>
            <input
              id={f.key}
              name={f.key}
              type={f.secret || f.key === "base_url" ? "password" : "text"}
              autoComplete="off"
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
              value={values[f.key] ?? ""}
              onChange={(e) => setValues({ ...values, [f.key]: e.target.value })}
              placeholder={f.value === "•••" ? "tersimpan (•••) - isi bila ingin mengubah" : ""}
            />
          </div>
        ))}
      </div>
      <div className="mt-4 flex gap-2">
        <button
          type="submit"
          disabled={saving}
          className="rounded-lg bg-zinc-900 px-4 py-2 text-sm font-semibold text-white hover:bg-zinc-800 disabled:opacity-50 dark:bg-white dark:text-zinc-900"
        >
          {saving ? "Menyimpan…" : "Simpan konfigurasi"}
        </button>
        {!isActive && (
          <button
            type="button"
            onClick={() => router.post("/admin/gateway/activate", { provider: provider.name })}
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"
          >
            Aktifkan
          </button>
        )}
      </div>
    </form>
  );
}

AdminGateways.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
