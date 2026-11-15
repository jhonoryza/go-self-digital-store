import { router } from "@inertiajs/react";
import { Check, ChevronDown } from "lucide-react";
import { useState, type FormEvent, type ReactNode } from "react";
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
  configured: boolean; // sudah punya kredensial tersimpan
  fields: Field[];
}

interface Props extends SharedProps {
  active: string;
  providers: Provider[];
}

// Konfigurasi gateway paygateway yang ringkas: daftar baris dengan radio
// untuk memilih gateway aktif; form kredensial hanya terbuka untuk satu
// provider yang dipilih (accordion), bukan semua sekaligus.
export default function AdminGateways({ active, providers }: Props) {
  const list = providers ?? [];
  // default terbuka: gateway yang sedang aktif (fallback: pertama)
  const [openName, setOpenName] = useState(active || list[0]?.name || "");
  const aktif = list.find((p) => p.name === active);

  return (
    <div className="max-w-3xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Gateway Pembayaran</h1>
        <p className="mt-1 text-sm text-zinc-500">
          Pilih gateway yang diaktifkan lewat lingkaran di kiri, lalu buka barisnya untuk mengatur
          kredensial. Secret tersimpan tampil tersamar (•••) - biarkan kosong bila tidak ingin mengubah.
        </p>
        {aktif ? (
          <p className="mt-3 text-sm">
            Aktif saat ini:{" "}
            <span className="rounded-full bg-brand-100 px-2.5 py-0.5 text-xs font-semibold text-brand-800 dark:bg-brand-950 dark:text-brand-300">
              {aktif.label}
            </span>
          </p>
        ) : (
          <p className="mt-3 text-sm text-zinc-500">Belum ada gateway dipilih - checkout memakai mode simulasi.</p>
        )}
      </div>

      <div className="divide-y divide-zinc-200 rounded-xl border border-zinc-200 bg-white dark:divide-zinc-800 dark:border-zinc-800 dark:bg-zinc-900">
        {list.map((p) => (
          <GatewayRow
            key={p.name}
            provider={p}
            isActive={active === p.name}
            open={openName === p.name}
            onToggle={() => setOpenName(openName === p.name ? "" : p.name)}
          />
        ))}
      </div>
    </div>
  );
}

function GatewayRow({
  provider,
  isActive,
  open,
  onToggle,
}: {
  provider: Provider;
  isActive: boolean;
  open: boolean;
  onToggle: () => void;
}) {
  const pilih = () => {
    if (!isActive) router.post("/admin/gateway/activate", { provider: provider.name });
  };

  return (
    <div>
      <div className={`flex items-center gap-3 p-4 transition-colors ${open ? "bg-zinc-50 dark:bg-zinc-800/40" : ""}`}>
        {/* radio aktivasi */}
        <button
          type="button"
          onClick={pilih}
          disabled={isActive}
          title={isActive ? "Sedang aktif" : `Aktifkan ${provider.label}`}
          aria-pressed={isActive}
          aria-label={`Aktifkan ${provider.label}`}
          className={`flex h-5 w-5 shrink-0 items-center justify-center rounded-full border-2 transition ${
            isActive
              ? "border-brand-600 bg-brand-600"
              : "border-zinc-300 hover:border-brand-500 dark:border-zinc-600"
          }`}
        >
          {isActive && <Check className="h-3 w-3 text-white" aria-hidden="true" />}
        </button>

        {/* baris pembuka form */}
        <button type="button" onClick={onToggle} aria-expanded={open} className="flex flex-1 items-center justify-between gap-3 text-left">
          <span className="min-w-0">
            <span className="font-semibold">{provider.label}</span>{" "}
            <span className="font-mono text-xs text-zinc-400">{provider.name}</span>
          </span>
          <span className="flex shrink-0 items-center gap-2">
            {provider.simulated && (
              <span className="rounded-full bg-zinc-200 px-2 py-0.5 text-xs font-semibold text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400">
                simulasi
              </span>
            )}
            {!provider.simulated && (
              <span
                className={`rounded-full px-2 py-0.5 text-xs font-semibold ${
                  provider.configured
                    ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400"
                    : "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-400"
                }`}
              >
                {provider.configured ? "terkonfigurasi" : "belum diatur"}
              </span>
            )}
            {isActive && (
              <span className="rounded-full bg-brand-100 px-2 py-0.5 text-xs font-bold text-brand-800 dark:bg-brand-950 dark:text-brand-300">
                AKTIF
              </span>
            )}
            <ChevronDown className={`h-4 w-4 text-zinc-400 transition-transform ${open ? "rotate-180" : ""}`} aria-hidden="true" />
          </span>
        </button>
      </div>

      {open && <GatewayForm provider={provider} />}
    </div>
  );
}

// GatewayForm form kredensial satu provider (field dinamis dari p.Fields()).
function GatewayForm({ provider }: { provider: Provider }) {
  const [values, setValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(provider.fields.map((f) => [f.key, f.value === "•••" ? "" : f.value])),
  );
  const [saving, setSaving] = useState(false);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setSaving(true);
    router.post(
      "/admin/gateway/config",
      { provider: provider.name, ...values },
      { onFinish: () => setSaving(false) },
    );
  };

  return (
    <form onSubmit={submit} className="border-t border-zinc-200 p-4 dark:border-zinc-800">
      <div className="grid gap-3 sm:grid-cols-2">
        {provider.fields.map((f) => (
          <div key={f.key}>
            <label htmlFor={`gw-${provider.name}-${f.key}`} className="mb-1 block text-xs font-medium">
              {f.label}
            </label>
            <input
              id={`gw-${provider.name}-${f.key}`}
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
      <div className="mt-4">
        <button
          type="submit"
          disabled={saving}
          className="rounded-lg bg-zinc-900 px-4 py-2 text-sm font-semibold text-white hover:bg-zinc-800 disabled:opacity-50 dark:bg-white dark:text-zinc-900"
        >
          {saving ? "Menyimpan…" : "Simpan konfigurasi"}
        </button>
      </div>
    </form>
  );
}

AdminGateways.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
