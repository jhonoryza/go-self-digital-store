import { router, useForm } from "@inertiajs/react";
import { ImagePlus, Loader2, RotateCcw, Search, X } from "lucide-react";
import { useRef, useState, type ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import FilterPanel from "../../components/FilterPanel";
import { useInfiniteList } from "../../lib/infinite";
import { rupiah, type ProductImage, type SharedProps } from "../../types";

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
  meta_title: string;
  meta_description: string;
  category?: string;
  tags?: string[];
}

interface Filters {
  q?: string;
  kategori?: string;
  type?: string;
  status?: string;
  sort?: string;
}

interface Props extends SharedProps {
  products: Product[];
  // peta product_id (string) -> daftar gambar dari bucket R2
  images: Record<string, ProductImage[]>;
  page: number;
  last_page: number;
  total: number;
  categories?: string[];
  filters?: Filters;
}

const SORTS: Record<string, string> = {
  terbaru: "Terbaru",
  terlama: "Terlama",
  termurah: "Harga termurah",
  termahal: "Harga termahal",
  nama: "Nama A-Z",
};

const MAX_IMAGES = 5;
const MAX_IMAGE_BYTES = 2 << 20;

export default function AdminProducts({ products: rawProducts, images, page, last_page, total, categories: rawCategories, filters = {} }: Props) {
  const [editing, setEditing] = useState<number | null>(null); // id produk yg form-nya terbuka
  const [showCreate, setShowCreate] = useState(false);
  const createForm = useRef<HTMLInputElement>(null);

  // daftar terakumulasi (infinite scroll) + toolbar filter
  const products = rawProducts ?? [];
  const categories = rawCategories ?? [];
  const adaFilter = Boolean(filters.q || filters.kategori || filters.type || filters.status);
  const nFilter = [filters.q, filters.kategori, filters.type, filters.status].filter(Boolean).length;
  const { items, busy, sentinelRef } = useInfiniteList(
    products,
    page,
    last_page,
    JSON.stringify(filters),
    ["products", "page", "last_page", "total"],
  );

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

      {/* toolbar cari + filter + urut: satu form GET, server yang menyaring */}
      <FilterPanel activeCount={nFilter} id="filter-produk">
      <form
        method="get"
        action="/admin/produk"
        className="rounded-xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900"
      >
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
          <div className="lg:col-span-2">
            <label htmlFor="q" className="mb-1 block text-xs font-medium text-zinc-500">Cari nama / slug</label>
            <input id="q" name="q" type="search" defaultValue={filters.q || ""} placeholder="Cari produk…"
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800" />
          </div>
          <div>
            <label htmlFor="kategori" className="mb-1 block text-xs font-medium text-zinc-500">Kategori</label>
            <select id="kategori" name="kategori" defaultValue={filters.kategori || ""}
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800">
              <option value="">Semua</option>
              {categories.map((c) => <option key={c} value={c}>{c}</option>)}
            </select>
          </div>
          <div>
            <label htmlFor="type" className="mb-1 block text-xs font-medium text-zinc-500">Jenis</label>
            <select id="type" name="type" defaultValue={filters.type || ""}
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800">
              <option value="">Semua</option>
              <option value="file">File</option>
              <option value="key">Key</option>
              <option value="link">Link</option>
            </select>
          </div>
          <div>
            <label htmlFor="status" className="mb-1 block text-xs font-medium text-zinc-500">Status</label>
            <select id="status" name="status" defaultValue={filters.status || ""}
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800">
              <option value="">Semua</option>
              <option value="aktif">Aktif</option>
              <option value="nonaktif">Nonaktif</option>
            </select>
          </div>
          <div>
            <label htmlFor="sort" className="mb-1 block text-xs font-medium text-zinc-500">Urutkan</label>
            <select id="sort" name="sort" defaultValue={filters.sort || "terbaru"}
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800">
              {Object.entries(SORTS).map(([v, label]) => <option key={v} value={v}>{label}</option>)}
            </select>
          </div>
        </div>
        <div className="mt-3 flex items-center gap-3">
          <button type="submit" className="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700">
            <Search className="h-4 w-4" aria-hidden="true" /> Terapkan
          </button>
          {adaFilter && (
            <a href="/admin/produk" className="inline-flex items-center gap-1.5 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200">
              <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" /> Reset
            </a>
          )}
          <span className="ml-auto text-sm text-zinc-500">{total} produk</span>
        </div>
      </form>
      </FilterPanel>

      {showCreate && (
        <ProductForm
          key="create"
          title="Produk baru"
          product={null}
          onDone={() => setShowCreate(false)}
        />
      )}

      <div className="space-y-4">
        {items.map((p) => (
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
                  {p.category && ` - ${p.category}`}
                  {p.type === "key" && ` - stok kunci: ${p.key_count}`}
                  {p.type === "file" && p.file_name && ` - ${p.file_name}`}
                  {(images[String(p.id)]?.length ?? 0) > 0 && ` - ${images[String(p.id)].length} gambar` }
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
                <ProductForm
                  key={p.id}
                  title={`Edit: ${p.title}`}
                  product={p}
                  images={images[String(p.id)] ?? []}
                  onDone={() => setEditing(null)}
                />
              </div>
            )}
          </div>
        ))}
        {items.length === 0 && (
          <p className="rounded-xl border border-dashed border-zinc-300 p-8 text-center text-zinc-500 dark:border-zinc-700">
            {adaFilter ? "Tidak ada produk yang cocok dengan filter." : "Belum ada produk - klik “Produk baru”."}
          </p>
        )}
        <div ref={sentinelRef} aria-hidden="true" className="h-1" />
        <div className="flex justify-center">
          {busy && (
            <p className="flex items-center gap-2 text-sm text-zinc-500">
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> Memuat produk lain…
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

// ProductForm form create/edit + tambah stok kunci (type=key).
function ProductForm({
  title,
  product,
  images = [],
  onDone,
}: {
  title: string;
  product: Product | null;
  images?: ProductImage[];
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
    meta_title: product?.meta_title ?? "",
    meta_description: product?.meta_description ?? "",
    kategori: product?.category ?? "",
    tags: product?.tags?.join(", ") ?? "",
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
          <label className="mb-1 block text-xs font-medium">Kategori (opsional)</label>
          <input id="kategori"
            className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.kategori} onChange={(e) => form.setData("kategori", e.target.value)}
            placeholder="mis. Template, E-Book, Preset" />
        </div>
        <div className="sm:col-span-2">
          <label className="mb-1 block text-xs font-medium">Tag (opsional, pisahkan dengan koma)</label>
          <input id="tags"
            className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.tags} onChange={(e) => form.setData("tags", e.target.value)}
            placeholder="mis. unduhan, siap-pakai, favorit" />
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
        <div className="sm:col-span-2 rounded-lg border border-dashed border-zinc-300 p-3 dark:border-zinc-700">
          <p className="mb-2 text-xs font-semibold text-zinc-500">SEO (opsional - kosong = otomatis)</p>
          <input
            placeholder="Meta title"
            value={form.data.meta_title}
            onChange={(e) => form.setData("meta_title", e.target.value)}
            className="mb-2 w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
          />
          <input
            placeholder="Meta description"
            value={form.data.meta_description}
            onChange={(e) => form.setData("meta_description", e.target.value)}
            className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
          />
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

      {product && <ImageManager productId={product.id} images={images} />}

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

// ImageManager kelola gambar produk di bucket R2 (maks 5, jpg/png/webp, 2 MB).
function ImageManager({ productId, images }: { productId: number; images: ProductImage[] }) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const remaining = MAX_IMAGES - images.length;

  const upload = () => {
    const files = inputRef.current?.files;
    if (!files || files.length === 0) return;
    if (files.length > remaining) {
      alert(`Sisa slot gambar ${remaining}.`);
      return;
    }
    for (const f of files) {
      if (f.size > MAX_IMAGE_BYTES) {
        alert(`${f.name} melebihi 2 MB.`);
        return;
      }
    }
    const fd = new FormData();
    for (const f of files) fd.append("images", f);
    setBusy(true);
    router.post(`/admin/produk/${productId}/images`, fd, {
      onSuccess: () => {
        if (inputRef.current) inputRef.current.value = "";
      },
      onFinish: () => setBusy(false),
    });
  };

  const remove = (img: ProductImage) => {
    router.post(`/admin/produk/${productId}/images/${img.id}/delete`);
  };

  return (
    <div className="mt-4 rounded-lg border border-dashed border-zinc-300 p-4 dark:border-zinc-700">
      <p className="mb-3 text-xs font-medium">
        Gambar produk (maks {MAX_IMAGES}) - saat ini {images.length}
      </p>
      {images.length > 0 && (
        <div className="mb-3 flex flex-wrap gap-2">
          {images.map((img) => (
            <div key={img.id} className="group relative">
              <img src={img.url} alt="" className="h-16 w-16 rounded-lg object-cover" />
              <button
                onClick={() => remove(img)}
                aria-label="Hapus gambar"
                className="absolute -right-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-red-600 text-white shadow hover:bg-red-700"
              >
                <X className="h-3 w-3" />
              </button>
            </div>
          ))}
        </div>
      )}
      {remaining > 0 ? (
        <div className="flex items-center gap-2">
          <input
            ref={inputRef}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            multiple
            className="block w-full text-xs text-zinc-500 file:mr-3 file:rounded-md file:border-0 file:bg-zinc-100 file:px-3 file:py-1.5 file:text-xs file:font-semibold hover:file:bg-zinc-200 dark:file:bg-zinc-800 dark:file:text-zinc-200"
            aria-label="Pilih gambar"
          />
          <button
            onClick={upload}
            disabled={busy}
            className="inline-flex shrink-0 items-center gap-1.5 rounded-lg bg-zinc-900 px-3 py-1.5 text-xs font-semibold text-white disabled:opacity-50 dark:bg-white dark:text-zinc-900"
          >
            <ImagePlus className="h-3.5 w-3.5" />
            {busy ? "Mengunggah…" : "Unggah"}
          </button>
        </div>
      ) : (
        <p className="text-xs text-zinc-500">Slot gambar penuh - hapus salah satu untuk mengganti.</p>
      )}
    </div>
  );
}
