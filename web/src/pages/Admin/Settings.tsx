import { router, usePage } from "@inertiajs/react";
import { Check, FileText, ImageIcon, Mail, Palette, Search } from "lucide-react";
import { useRef, useState, type ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import type { SharedProps } from "../../types";

interface Props extends SharedProps {
  smtp: { host: string; port: string; user: string; from: string; has_pass: string };
  brand: { name: string; logo: string };
  content: { privacy: string; terms: string };
  seo: Record<string, string>;
  theme: string;
  images_enabled: boolean;
}

// Daftar tema tampilan toko; warna cermin override di web/src/css/app.css.
const THEMES = [
  { id: "", label: "Bawaan", desc: "Hijau toko - tampilan default", swatch: "#277758", bg: "#f0f9f4" },
  { id: "brutalism", label: "Brutalism", desc: "Sudut tegas, garis tebal, bayangan offset", swatch: "#4d7c0f", bg: "#f7fee7" },
  { id: "neon", label: "Neon Cyber", desc: "Fuchsia menyala dengan efek glow", swatch: "#c026d3", bg: "#fdf4ff" },
  { id: "pastel", label: "Pastel Lembut", desc: "Violet kalem, sudut ekstra membulat", swatch: "#7c3aed", bg: "#f5f3ff" },
  { id: "terminal", label: "Retro Terminal", desc: "Hijau fosfor layar CRT", swatch: "#059669", bg: "#ecfdf5" },
  { id: "ocean", label: "Samudra", desc: "Biru klasik aplikasi SaaS", swatch: "#2563eb", bg: "#eff6ff" },
  { id: "coss", label: "Coss UI", desc: "Indigo Cal.com - tenang dan aksesibel", swatch: "#3e63dd", bg: "#f1f2fb" },
  { id: "align", label: "Align UI", desc: "Biru cerah AlignUI (primary default)", swatch: "#1570ef", bg: "#eff8ff" },
  { id: "daisy", label: "daisyUI", desc: "Indigo vivid tema light daisyUI 5", swatch: "#0038f7", bg: "#dff1ff" },
  { id: "hero", label: "HeroUI", desc: "Biru HeroUI (dahulu NextUI)", swatch: "#005bc4", bg: "#e6f1fd" },
  { id: "ant", label: "Ant Design", desc: "Daybreak Blue khas antd", swatch: "#0958d9", bg: "#e6f4ff" },
] as const;

// Halaman Setting: SMTP runtime, brand (nama + logo), konten kebijakan,
// dan SEO per halaman. Semua tersimpan di tabel settings, aktif tanpa deploy.
export default function AdminSettings({ smtp, brand, content, seo, theme, images_enabled }: Props) {
  const logoRef = useRef<HTMLInputElement>(null);
  const [seoData, setSeoData] = useState<Record<string, string>>({
    "seo.home.title": seo["seo.home.title"] ?? "",
    "seo.home.description": seo["seo.home.description"] ?? "",
    "seo.products.title": seo["seo.products.title"] ?? "",
    "seo.products.description": seo["seo.products.description"] ?? "",
    "seo.product.title": seo["seo.product.title"] ?? "",
    "seo.product.description": seo["seo.product.description"] ?? "",
  });
  const setSeo = (k: string, v: string) => setSeoData((prev) => ({ ...prev, [k]: v }));

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Pengaturan</h1>

      {/* SMTP */}
      <section className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <h2 className="mb-4 flex items-center gap-2 font-semibold">
          <Mail className="h-4 w-4 text-brand-600 dark:text-brand-400" /> SMTP (email keluar)
        </h2>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            router.post("/admin/setting/smtp", new FormData(e.currentTarget));
          }}
          className="grid gap-4 sm:grid-cols-2"
        >
          <Field label="Host" name="host" defaultValue={smtp.host} />
          <Field label="Port" name="port" defaultValue={smtp.port} />
          <Field label="User" name="user" defaultValue={smtp.user} />
          <Field label="Password (kosongkan bila tetap)" name="pass" type="password" />
          <Field label="Dari (From)" name="from" defaultValue={smtp.from} />
          <div className="flex items-end gap-2 sm:col-span-2">
            <button className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700">
              Simpan SMTP
            </button>
            <button
              type="button"
              onClick={() => router.post("/admin/setting/smtp/test")}
              className="rounded-lg border border-zinc-300 px-4 py-2 text-sm font-semibold hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
            >
              Kirim email uji
            </button>
          </div>
        </form>
      </section>

      {/* Brand */}
      <section className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <h2 className="mb-4 flex items-center gap-2 font-semibold">
          <ImageIcon className="h-4 w-4 text-brand-600 dark:text-brand-400" /> Brand (nama + logo)
        </h2>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            router.post("/admin/setting/brand", new FormData(e.currentTarget));
          }}
          className="grid items-start gap-4 sm:grid-cols-[1fr_auto]"
        >
          <div className="space-y-3">
            <Field label="Nama toko (menimpa default env)" name="name" defaultValue={brand.name} />
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" name="remove_logo" value="1" />
              Hapus logo saat ini
            </label>
            {!images_enabled && (
              <p className="text-xs text-brass-600 dark:text-brass-400">
                Unggah logo butuh penyimpanan R2 aktif (STORE_R2_*).
              </p>
            )}
          </div>
          <div className="flex flex-col items-center gap-2">
            {brand.logo && <img src={brand.logo} alt="Logo" className="h-16 w-16 rounded-lg object-contain" />}
            <input
              ref={logoRef}
              type="file"
              name="logo"
              accept="image/jpeg,image/png,image/webp,image/svg+xml"
              disabled={!images_enabled}
              className="block w-full text-xs text-zinc-500 file:mr-3 file:rounded-md file:border-0 file:bg-zinc-100 file:px-3 file:py-1.5 file:text-xs file:font-semibold dark:file:bg-zinc-800 dark:file:text-zinc-200"
            />
            <button className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700">
              Simpan brand
            </button>
          </div>
        </form>
      </section>

      {/* Tema tampilan */}
      <section className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <h2 className="mb-1 flex items-center gap-2 font-semibold">
          <Palette className="h-4 w-4 text-brand-600 dark:text-brand-400" /> Tema tampilan toko
        </h2>
        <p className="mb-4 text-sm text-zinc-500">
          Berlaku untuk seluruh halaman pengunjung; pengunjung tetap bisa memilih mode gelap/terang.
        </p>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {THEMES.map((t) => {
            const aktif = (theme || "") === t.id;
            return (
              <button
                key={t.id}
                type="button"
                onClick={() => router.post("/admin/setting/theme", { theme: t.id })}
                aria-pressed={aktif}
                className={`rounded-xl border-2 p-3 text-left transition ${
                  aktif
                    ? "border-brand-600 bg-brand-50 dark:border-brand-500 dark:bg-brand-950/40"
                    : "border-zinc-200 hover:border-brand-400 dark:border-zinc-800 dark:hover:border-brand-600"
                }`}
              >
                <span
                  className="mb-3 flex h-16 items-end gap-1.5 rounded-lg border border-black/10 p-2 dark:border-white/10"
                  style={{ backgroundColor: t.bg }}
                >
                  <span className="h-2 w-10 rounded-full" style={{ backgroundColor: t.swatch }} />
                  <span className="h-2 w-6 rounded-full opacity-50" style={{ backgroundColor: t.swatch }} />
                  <span className="ml-auto h-5 w-12 rounded-md" style={{ backgroundColor: t.swatch }} />
                </span>
                <span className="flex items-center justify-between">
                  <span className="font-semibold">{t.label}</span>
                  {aktif && (
                    <span className="flex h-5 w-5 items-center justify-center rounded-full bg-brand-600 text-white">
                      <Check className="h-3 w-3" aria-hidden="true" />
                    </span>
                  )}
                </span>
                <span className="mt-0.5 block text-xs text-zinc-500">{t.desc}</span>
              </button>
            );
          })}
        </div>
      </section>

      {/* Konten kebijakan */}
      <section className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <h2 className="mb-4 flex items-center gap-2 font-semibold">
          <FileText className="h-4 w-4 text-brand-600 dark:text-brand-400" /> Konten privasi & ketentuan
        </h2>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            router.post("/admin/setting/content", new FormData(e.currentTarget));
          }}
          className="grid gap-4 sm:grid-cols-2"
        >
          <div>
            <label className="mb-1 block text-sm font-medium">Isi halaman Privasi</label>
            <textarea
              name="privacy"
              rows={8}
              defaultValue={content.privacy}
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            />
            <p className="mt-1 text-xs text-zinc-500">Kosong = pakai konten bawaan.</p>
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium">Isi halaman Ketentuan</label>
            <textarea
              name="terms"
              rows={8}
              defaultValue={content.terms}
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            />
            <p className="mt-1 text-xs text-zinc-500">Kosong = pakai konten bawaan.</p>
          </div>
          <div className="sm:col-span-2">
            <button className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700">
              Simpan konten
            </button>
          </div>
        </form>
      </section>

      {/* SEO */}
      <section className="rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900">
        <h2 className="mb-4 flex items-center gap-2 font-semibold">
          <Search className="h-4 w-4 text-brand-600 dark:text-brand-400" /> SEO per halaman
        </h2>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            router.post("/admin/setting/seo", seoData);
          }}
          className="space-y-4"
        >
          <SeoPair
            label="Beranda"
            titleName="seo.home.title"
            descName="seo.home.description"
            title={seoData["seo.home.title"]}
            desc={seoData["seo.home.description"]}
            onChange={setSeo}
          />
          <SeoPair
            label="Katalog produk"
            titleName="seo.products.title"
            descName="seo.products.description"
            title={seoData["seo.products.title"]}
            desc={seoData["seo.products.description"]}
            onChange={setSeo}
          />
          <div className="rounded-lg border border-dashed border-zinc-300 p-4 dark:border-zinc-700">
            <p className="mb-2 text-sm font-semibold">Detail produk (template)</p>
            <p className="mb-2 text-xs text-zinc-500">
              Variabel tersedia: <code className="font-mono">{"{title}"}</code>,{" "}
              <code className="font-mono">{"{price}"}</code>. Kosong = otomatis dari judul produk.
            </p>
            <SeoPair
              label=""
              titleName="seo.product.title"
              descName="seo.product.description"
              title={seoData["seo.product.title"]}
              desc={seoData["seo.product.description"]}
              onChange={setSeo}
              titlePlaceholder="Beli {title} - Rp{price}"
              descPlaceholder="{title}: {description}"
            />
          </div>
          <button className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700">
            Simpan SEO
          </button>
        </form>
      </section>
    </div>
  );
}

AdminSettings.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;

function Field({
  label,
  name,
  defaultValue,
  type = "text",
}: {
  label: string;
  name: string;
  defaultValue?: string;
  type?: string;
}) {
  return (
    <div>
      <label htmlFor={name} className="mb-1 block text-sm font-medium">
        {label}
      </label>
      <input
        id={name}
        name={name}
        type={type}
        defaultValue={defaultValue}
        className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
      />
    </div>
  );
}

function SeoPair({
  label,
  titleName,
  descName,
  title,
  desc,
  onChange,
  titlePlaceholder,
  descPlaceholder,
}: {
  label: string;
  titleName: string;
  descName: string;
  title: string;
  desc: string;
  onChange: (k: string, v: string) => void;
  titlePlaceholder?: string;
  descPlaceholder?: string;
}) {
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {label && <p className="text-sm font-semibold sm:col-span-2">{label}</p>}
      <div>
        <label className="mb-1 block text-xs font-medium text-zinc-500">Meta title</label>
        <input
          value={title}
          onChange={(e) => onChange(titleName, e.target.value)}
          placeholder={titlePlaceholder}
          className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
        />
      </div>
      <div>
        <label className="mb-1 block text-xs font-medium text-zinc-500">Meta description</label>
        <input
          value={desc}
          onChange={(e) => onChange(descName, e.target.value)}
          placeholder={descPlaceholder}
          className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
        />
      </div>
    </div>
  );
}
