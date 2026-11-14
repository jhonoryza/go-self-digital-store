import { router, useForm } from "@inertiajs/react";
import { useRef, useState, type ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import { rupiah, type SharedProps } from "../../types";

interface Product {
  id: number;
  slug: string;
  title: string;
  description: string;
  price: number;
  type: string;
  featured: boolean;
  active: boolean;
  file_path: string;
  file_name: string;
  ext_url: string;
  key_count: number;
}

interface Props extends SharedProps {
  products: Product[];
}

export default function AdminProducts({ products }: Props) {
  const [editing, setEditing] = useState<number | null>(null); // id produk yg form-nya terbuka
  const [showCreate, setShowCreate] = useState(false);
  const createForm = useRef<HTMLInputElement>(null);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Produk</h1>
        <button
          onClick={() => setShowCreate(!showCreate)}
          className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"
        >
          {showCreate ? "Tutup" : "+ Produk baru"}
        </button>
      </div>

      {showCreate && (
        <ProductForm
          key="create"
          title="Produk baru"
          product={null}
          onDone={() => setShowCreate(false)}
        />
      )}

      <div className="space-y-4">
        {(products ?? []).map((p) => (
          <div key={p.id} className="rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
            <div className="flex flex-wrap items-center justify-between gap-3 p-4">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-semibold">{p.title}</span>
                  {!p.active && <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-semibold text-red-700 dark:bg-red-950 dark:text-red-400">nonaktif</span>}
                  {p.featured && <span className="rounded-full bg-brass-400/20 px-2 py-0.5 text-xs font-semibold text-brass-600 dark:text-brass-400">unggulan</span>}
                </div>
                <p className="mt-0.5 text-xs text-zinc-500">
                  <span className="font-mono">{p.slug}</span> - {p.type} - {rupiah(p.price)}
                  {p.type === "key" && ` - stok kunci: ${p.key_count}`}
                  {p.type === "file" && p.file_name && ` - ${p.file_name}`}
                </p>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => router.post(`/admin/produk/${p.id}/toggle`)}
                  className="rounded-md border border-zinc-300 px-2.5 py-1 text-xs font-semibold hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800"
                >
                  {p.active ? "Nonaktifkan" : "Aktifkan"}
                </button>
                <button
                  onClick={() => setEditing(editing === p.id ? null : p.id)}
                  className="rounded-md bg-zinc-900 px-2.5 py-1 text-xs font-semibold text-white dark:bg-white dark:text-zinc-900"
                >
                  {editing === p.id ? "Tutup" : "Edit"}
                </button>
              </div>
            </div>
            {editing === p.id && (
              <div className="border-t border-zinc-200 p-4 dark:border-zinc-800">
                <ProductForm key={p.id} title={`Edit: ${p.title}`} product={p} onDone={() => setEditing(null)} />
              </div>
            )}
          </div>
        ))}
        {(products ?? []).length === 0 && (
          <p className="rounded-xl border border-dashed border-zinc-300 p-8 text-center text-zinc-500 dark:border-zinc-700">
            Belum ada produk - klik “Produk baru”.
          </p>
        )}
      </div>
    </div>
  );
}

// ProductForm form create/edit + tambah stok kunci (type=key).
function ProductForm({
  title,
  product,
  onDone,
}: {
  title: string;
  product: Product | null;
  onDone: () => void;
}) {
  const form = useForm({
    slug: product?.slug ?? "",
    title: product?.title ?? "",
    description: product?.description ?? "",
    price: product ? String(product.price) : "",
    type: product?.type ?? "file",
    ext_url: product?.ext_url ?? "",
    featured: product?.featured ? "1" : "0",
    active: product?.active ? "1" : "1",
    file: null as File | null,
  });
  const keysForm = useForm({ keys: "" });
  const fileRef = useRef<HTMLInputElement>(null);
  const url = product ? `/admin/produk/${product.id}` : "/admin/produk";

  return (
    <div className="space-y-4">
      <h3 className="font-semibold">{title}</h3>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          form.post(url, { onSuccess: onDone });
        }}
        className="grid gap-4 sm:grid-cols-2"
      >
        <div>
          <label className="mb-1 block text-xs font-medium">Slug (URL)</label>
          <input id="slug" className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.slug} onChange={(e) => form.setData("slug", e.target.value)} required />
          {form.errors.slug && <p className="mt-1 text-xs text-red-600">{form.errors.slug}</p>}
        </div>
        <div>
          <label className="mb-1 block text-xs font-medium">Judul</label>
          <input id="title" className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.title} onChange={(e) => form.setData("title", e.target.value)} required />
          {form.errors.title && <p className="mt-1 text-xs text-red-600">{form.errors.title}</p>}
        </div>
        <div className="sm:col-span-2">
          <label className="mb-1 block text-xs font-medium">Deskripsi</label>
          <textarea id="description" rows={3}
            className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.description} onChange={(e) => form.setData("description", e.target.value)} />
        </div>
        <div>
          <label className="mb-1 block text-xs font-medium">Harga (rupiah)</label>
          <input id="price" type="number" min="1"
            className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.price} onChange={(e) => form.setData("price", e.target.value)} required />
          {form.errors.price && <p className="mt-1 text-xs text-red-600">{form.errors.price}</p>}
        </div>
        <div>
          <label className="mb-1 block text-xs font-medium">Jenis</label>
          <select id="type"
            className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.type} onChange={(e) => form.setData("type", e.target.value)}>
            <option value="file">File (unduhan)</option>
            <option value="key">Kunci lisensi</option>
            <option value="link">Tautan eksternal</option>
          </select>
        </div>
        {form.data.type === "file" && (
          <div>
            <label className="mb-1 block text-xs font-medium">
              Berkas {product?.file_name && <span className="text-zinc-400">(saat ini: {product.file_name})</span>}
            </label>
            <input ref={fileRef} type="file"
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
              onChange={(e) => form.setData("file", e.target.files?.[0] ?? null)} />
            {form.errors.file && <p className="mt-1 text-xs text-red-600">{form.errors.file}</p>}
          </div>
        )}
        {form.data.type === "link" && (
          <div>
            <label className="mb-1 block text-xs font-medium">Tautan eksternal</label>
            <input id="ext_url" type="url"
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
              value={form.data.ext_url} onChange={(e) => form.setData("ext_url", e.target.value)} />
            {form.errors.ext_url && <p className="mt-1 text-xs text-red-600">{form.errors.ext_url}</p>}
          </div>
        )}
        <div className="flex items-end gap-6">
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={form.data.featured === "1"}
              onChange={(e) => form.setData("featured", e.target.checked ? "1" : "0")} />
            Unggulan
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={form.data.active === "1"}
              onChange={(e) => form.setData("active", e.target.checked ? "1" : "0")} />
            Aktif
          </label>
        </div>
        <div className="flex gap-2 sm:col-span-2">
          <button type="submit" disabled={form.processing}
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50">
            {form.processing ? "Menyimpan…" : "Simpan"}
          </button>
          <button type="button" onClick={onDone}
            className="rounded-lg border border-zinc-300 px-4 py-2 text-sm font-semibold hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800">
            Batal
          </button>
        </div>
      </form>

      {product?.type === "key" && (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            keysForm.post(`/admin/produk/${product.id}/keys`, { onSuccess: () => keysForm.reset() });
          }}
          className="rounded-lg border border-dashed border-zinc-300 p-4 dark:border-zinc-700"
        >
          <label className="mb-1 block text-xs font-medium">
            Tambah stok kunci (satu kunci per baris) - saat ini {product.key_count}
          </label>
          <textarea id="keys" rows={3}
            className="w-full rounded-lg border-zinc-300 font-mono text-xs dark:border-zinc-700 dark:bg-zinc-800"
            value={keysForm.data.keys} onChange={(e) => keysForm.setData("keys", e.target.value)}
            placeholder={"KEY-0001\nKEY-0002"} />
          <button type="submit" disabled={keysForm.processing}
            className="mt-2 rounded-lg bg-zinc-900 px-3 py-1.5 text-xs font-semibold text-white dark:bg-white dark:text-zinc-900">
            {keysForm.processing ? "Menambah…" : "Tambah kunci"}
          </button>
        </form>
      )}
    </div>
  );
}

AdminProducts.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
