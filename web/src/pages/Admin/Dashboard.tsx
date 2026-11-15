import { Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import { rupiah, type SharedProps } from "../../types";

interface Stats {
  revenue: number;
  orders_paid: number;
  orders_pending: number;
  users: number;
  tickets_open: number;
  mail_error: number;
  license_free: number;
  license_sold: number;
}

interface Props extends SharedProps {
  stats: Stats;
  periode: string; // "" = semua | bulanan | mingguan | harian
  series: { label: string; value: number }[]; // pendapatan lunas per bucket
  pie: { status: string; count: number }[]; // distribusi status order
}

const PERIODES: [string, string][] = [
  ["", "Semua"],
  ["bulanan", "Bulanan"],
  ["mingguan", "Mingguan"],
  ["harian", "Harian"],
];

const WARNA_PIE: Record<string, string> = {
  paid: "var(--color-brand-600)",
  pending: "#f59e0b",
};
const LABEL_PIE: Record<string, string> = {
  paid: "Lunas",
  pending: "Pending",
};

// GrafikGaris: area + garis pendapatan per bucket (SVG, tanpa dependensi).
function GrafikGaris({ data }: { data: { label: string; value: number }[] }) {
  const W = 600;
  const H = 220;
  const pad = { l: 6, r: 6, t: 12, b: 22 };
  const max = Math.max(1, ...data.map((d) => d.value));
  const x = (i: number) => pad.l + (i * (W - pad.l - pad.r)) / Math.max(1, data.length - 1);
  const y = (v: number) => H - pad.b - (v / max) * (H - pad.t - pad.b);
  const pts = data.map((d, i) => `${x(i)},${y(d.value)}`).join(" ");
  const langkahLabel = Math.max(1, Math.ceil(data.length / 10));

  return (
    <svg viewBox={`0 0 ${W} ${H}`} className="h-56 w-full" role="img" aria-label="Grafik pendapatan lunas per periode">
      {[0.25, 0.5, 0.75, 1].map((f) => (
        <line key={f} x1={pad.l} x2={W - pad.r} y1={y(max * f)} y2={y(max * f)} stroke="currentColor" className="text-zinc-200 dark:text-zinc-800" strokeWidth="1" />
      ))}
      <polygon points={`${x(0)},${H - pad.b} ${pts} ${x(data.length - 1)},${H - pad.b}`} fill="var(--color-brand-500)" opacity="0.15" />
      <polyline points={pts} fill="none" stroke="var(--color-brand-600)" strokeWidth="2.5" strokeLinejoin="round" />
      {data.map((d, i) => (
        <circle key={i} cx={x(i)} cy={y(d.value)} r="2.5" fill="var(--color-brand-600)" />
      ))}
      {data.map((d, i) =>
        i % langkahLabel === 0 ? (
          <text key={`l${i}`} x={x(i)} y={H - 6} textAnchor="middle" fontSize="9" className="fill-zinc-500">
            {d.label}
          </text>
        ) : null,
      )}
      <text x={pad.l} y={pad.t + 2} fontSize="9" className="fill-zinc-400">
        maks {rupiah(max)}
      </text>
    </svg>
  );
}

// GrafikPie: donat distribusi status order (SVG stroke-dasharray).
function GrafikPie({ data }: { data: { label: string; value: number; color: string }[] }) {
  const total = data.reduce((sum, d) => sum + d.value, 0);
  const R = 56;
  const C = 2 * Math.PI * R;
  let lewat = 0;
  return (
    <div className="flex items-center gap-6">
      <svg viewBox="0 0 140 140" className="h-40 w-40 shrink-0" role="img" aria-label="Distribusi status order">
        <circle cx="70" cy="70" r={R} fill="none" strokeWidth="24" className="stroke-zinc-100 dark:stroke-zinc-800" />
        {total > 0 &&
          data.map((d, i) => {
            const len = (d.value / total) * C;
            const seg = (
              <circle
                key={i}
                cx="70"
                cy="70"
                r={R}
                fill="none"
                strokeWidth="24"
                stroke={d.color}
                strokeDasharray={`${len} ${C - len}`}
                strokeDashoffset={-lewat}
                transform="rotate(-90 70 70)"
              />
            );
            lewat += len;
            return seg;
          })}
        <text x="70" y="66" textAnchor="middle" fontSize="20" fontWeight="700" className="fill-zinc-900 dark:fill-zinc-100">
          {total}
        </text>
        <text x="70" y="84" textAnchor="middle" fontSize="9" className="fill-zinc-500">
          order
        </text>
      </svg>
      <ul className="space-y-2 text-sm">
        {data.map((d) => (
          <li key={d.label} className="flex items-center gap-2">
            <span className="h-3 w-3 rounded-full" style={{ backgroundColor: d.color }} aria-hidden="true" />
            <span>{d.label}</span>
            <span className="font-semibold">{d.value}</span>
            <span className="text-xs text-zinc-500">({total > 0 ? Math.round((d.value / total) * 100) : 0}%)</span>
          </li>
        ))}
        {total === 0 && <li className="text-zinc-500">Belum ada order dalam periode ini.</li>}
      </ul>
    </div>
  );
}

export default function AdminDashboard({ stats, periode, series, pie }: Props) {
  const cards: [string, string, string][] = [
    ["Pendapatan lunas", rupiah(stats.revenue), "text-brand-700 dark:text-brand-400"],
    ["Order lunas", String(stats.orders_paid), ""],
    ["Order pending", String(stats.orders_pending), stats.orders_pending > 0 ? "text-amber-600" : ""],
    ["Pengguna", String(stats.users), ""],
    ["Tiket terbuka", String(stats.tickets_open), stats.tickets_open > 0 ? "text-amber-600" : ""],
    ["Email gagal", String(stats.mail_error), stats.mail_error > 0 ? "text-red-600" : ""],
    ["Stok kunci tersisa", String(stats.license_free), stats.license_free === 0 ? "text-red-600" : ""],
    ["Kunci terjual", String(stats.license_sold), ""],
  ];
  const pieData = pie.map((d) => ({
    label: LABEL_PIE[d.status] ?? d.status,
    value: d.count,
    color: WARNA_PIE[d.status] ?? "#a1a1aa",
  }));

  return (
    <div className="space-y-8">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold">Konsol Admin</h1>
        {/* filter periode: kartu order + kedua grafik */}
        <nav aria-label="Filter periode" className="flex rounded-lg border border-zinc-300 p-0.5 text-sm dark:border-zinc-700">
          {PERIODES.map(([v, label]) => (
            <Link
              key={v || "all"}
              href={v ? `/admin?periode=${v}` : "/admin"}
              aria-current={(periode || "") === v ? "true" : undefined}
              className={`rounded-md px-3 py-1.5 font-medium ${
                (periode || "") === v
                  ? "bg-brand-600 text-white"
                  : "text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
              }`}
            >
              {label}
            </Link>
          ))}
        </nav>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {cards.map(([label, value, cls]) => (
          <div key={label} className="rounded-xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
            <p className="text-sm text-zinc-500">{label}</p>
            <p className={`mt-1 text-2xl font-extrabold ${cls}`}>{value}</p>
          </div>
        ))}
      </div>

      <div className="grid gap-6 lg:grid-cols-5">
        <section className="rounded-xl border border-zinc-200 bg-white p-5 lg:col-span-3 dark:border-zinc-800 dark:bg-zinc-900">
          <h2 className="mb-3 font-semibold">Pendapatan lunas</h2>
          <GrafikGaris data={series} />
        </section>
        <section className="rounded-xl border border-zinc-200 bg-white p-5 lg:col-span-2 dark:border-zinc-800 dark:bg-zinc-900">
          <h2 className="mb-3 font-semibold">Status order</h2>
          <GrafikPie data={pieData} />
        </section>
      </div>
      <p className="text-xs text-zinc-500">
        Kartu Pendapatan/Order mengikuti filter periode; kartu Pengguna, Tiket, dan Email selalu total.
        Tindakan lanjutan (tandai lunas, kirim ulang) di tab{" "}
        <Link href="/admin/orders" className="text-brand-600 hover:underline dark:text-brand-400">
          Orders
        </Link>
        .
      </p>
    </div>
  );
}

AdminDashboard.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
