import { Head, Link, router, useForm, usePage } from "@inertiajs/react";
import { Check, Share2, ShoppingCart, Star } from "lucide-react";
import { useState, type ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import ProductThumb from "../../components/ProductThumb";
import ProductTypeIcon from "../../components/ProductTypeIcon";
import { addToCart } from "../../lib/cart";
import { rupiah, type Product, type ProductImage, type SharedProps } from "../../types";

interface ReviewItem {
  rating: number;
  body: string;
  name: string;
  created_at: string;
}

interface Props extends SharedProps {
  product: Product;
  images: ProductImage[];
  related?: Product[]; // produk satu kategori (maks 4)
  reviews?: ReviewItem[]; // ulasan pembeli (terbaru dulu)
  rating_avg?: number;
  rating_count?: number;
  can_review?: boolean; // pembeli lunas yang belum mengulas
  share_url?: string; // URL absolut utk tombol bagikan (WhatsApp)
  seo?: { "seo.product.title"?: string; "seo.product.description"?: string };
}

// Bintang rating: n terisi (kuning), sisanya outline.
function Bintang({ n, className = "h-4 w-4" }: { n: number; className?: string }) {
  return (
    <span className="flex gap-0.5" aria-label={`${n} dari 5 bintang`}>
      {[1, 2, 3, 4, 5].map((i) => (
        <Star
          key={i}
          className={`${className} ${i <= n ? "fill-amber-400 text-amber-400" : "text-zinc-300 dark:text-zinc-700"}`}
          aria-hidden="true"
        />
      ))}
    </span>
  );
}

export default function ProductDetail({
  product,
  images: rawImages,
  related: rawRelated,
  reviews: rawReviews,
  rating_avg = 0,
  rating_count = 0,
  can_review,
  share_url,
  seo,
}: Props) {
  const { auth_user } = usePage<SharedProps>().props;
  const related = rawRelated ?? [];
  const reviews = rawReviews ?? [];
  const habis = product.type === "key" && product.key_stock !== undefined && product.key_stock === 0;
  const reviewForm = useForm({ rating: 0, body: "" });
  const images = rawImages ?? []; // props nil dihilangkan gonertia
  const { app_name } = usePage<SharedProps>().props;
  const metaTitle = (seo?.["seo.product.title"] || "{title} - {price}")
    .replace("{title}", product.title)
    .replace("{price}", rupiah(product.price));
  const metaDesc = (seo?.["seo.product.description"] || "{title}: {description}")
    .replace("{title}", product.title)
    .replace("{description}", product.description.slice(0, 150));
  const [added, setAdded] = useState(false);
  const [selected, setSelected] = useState(0);
  const add = () => {
    addToCart(product);
    setAdded(true);
    setTimeout(() => setAdded(false), 1500);
  };
  const main = images[selected]?.url;
  return (
    <>
      <Head>
        <title>{product.meta_title || metaTitle}</title>
        <meta name="description" content={product.meta_description || metaDesc} />
      </Head>
      <section className="mx-auto max-w-4xl px-4 py-12">
      <Link href="/produk" className="mb-6 inline-block text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200">
        ← Semua produk
      </Link>
      <div className="rounded-xl border border-zinc-200 bg-white p-8 dark:border-zinc-800 dark:bg-zinc-900">
        {/* Galeri: gambar utama + thumb (maks 5); tanpa gambar tampilkan ikon jenis */}
        {main ? (
          <div>
            <img
              src={main}
              alt={product.title}
              className="max-h-80 w-full rounded-lg object-contain"
            />
            {images.length > 1 && (
              <div className="mt-3 flex gap-2">
                {images.map((img, i) => (
                  <button
                    key={img.id}
                    onClick={() => setSelected(i)}
                    aria-label={`Gambar ${i + 1}`}
                    className={`overflow-hidden rounded-lg border-2 ${
                      i === selected ? "border-brand-600" : "border-transparent opacity-70 hover:opacity-100"
                    }`}
                  >
                    <img src={img.url} alt="" className="h-16 w-16 object-cover" />
                  </button>
                ))}
              </div>
            )}
          </div>
        ) : (
          <div className="mb-4 flex h-32 items-center justify-center rounded-lg bg-brand-50 text-5xl dark:bg-brand-950">
            <ProductTypeIcon type={product.type} className="h-6 w-6" />
          </div>
        )}
        <h1 className="text-3xl font-bold">{product.title}</h1>
        {(product.category || (product.tags && product.tags.length > 0)) && (
          <div className="mt-3 flex flex-wrap items-center gap-2">
            {product.category && (
              <Link
                href={`/produk?kategori=${encodeURIComponent(product.category)}`}
                className="rounded-full bg-brand-50 px-2.5 py-0.5 text-xs font-semibold text-brand-700 hover:bg-brand-100 dark:bg-brand-950 dark:text-brand-300 dark:hover:bg-brand-900"
              >
                {product.category}
              </Link>
            )}
            {product.tags?.map((t) => (
              <Link
                key={t}
                href={`/produk?tag=${encodeURIComponent(t)}`}
                className="rounded-full bg-zinc-100 px-2.5 py-0.5 text-xs text-zinc-600 hover:bg-zinc-200 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-700"
              >
                #{t}
              </Link>
            ))}
          </div>
        )}
        <p className="mt-4 whitespace-pre-wrap leading-relaxed text-zinc-600 dark:text-zinc-400">
          {product.description}
        </p>
        <div className="mt-8 flex flex-wrap items-center justify-between gap-4 border-t border-zinc-200 pt-6 dark:border-zinc-800">
          <span className="text-2xl font-extrabold text-brand-700 dark:text-brand-300">{rupiah(product.price)}</span>
          <div className="flex items-center gap-3">
            {product.type === "key" && product.key_stock !== undefined && (
              <span className={`text-sm font-medium ${habis ? "text-red-600" : "text-zinc-500"}`}>
                Stok kunci: {product.key_stock}
              </span>
            )}
            <button
              onClick={add}
              disabled={habis}
              className="inline-flex items-center gap-1.5 rounded-lg border border-brand-600 px-4 py-2.5 text-sm font-semibold text-brand-700 hover:bg-brand-50 disabled:cursor-not-allowed disabled:opacity-40 dark:text-brand-400 dark:hover:bg-brand-950 dark:text-brand-300"
            >
              {added ? <Check className="h-4 w-4" /> : <ShoppingCart className="h-4 w-4" />}
              {added ? "Ditambahkan" : "Keranjang"}
            </button>
            <button
              onClick={() => {
                addToCart(product);
                router.visit("/checkout");
              }}
              disabled={habis}
              className="rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {habis ? "Stok habis" : "Beli Sekarang"}
            </button>
            {share_url && (
              <a
                href={`https://wa.me/?text=${encodeURIComponent(product.title + " - " + rupiah(product.price) + "\n" + share_url)}`}
                target="_blank"
                rel="noopener noreferrer"
                aria-label="Bagikan via WhatsApp"
                title="Bagikan via WhatsApp"
                className="flex h-10 w-10 items-center justify-center rounded-lg border border-zinc-300 text-zinc-600 hover:border-brand-400 hover:text-brand-600 dark:border-zinc-700 dark:text-zinc-400"
              >
                <Share2 className="h-4 w-4" />
              </a>
            )}
          </div>
        </div>
      </div>

      <section className="mt-12">
        <h2 className="font-semibold">Ulasan pembeli</h2>
        {rating_count > 0 && (
          <p className="mt-1 flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
            <Bintang n={Math.round(rating_avg)} />
            <span className="font-semibold">{rating_avg.toFixed(1)}</span>
            <span>dari {rating_count} ulasan</span>
          </p>
        )}

        {auth_user && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              reviewForm.post(`/produk/${product.slug}/ulasan`, { onSuccess: () => reviewForm.reset() });
            }}
            className="mt-4 rounded-xl border border-zinc-200 p-4 dark:border-zinc-800"
          >
            <div className="flex items-center gap-1" role="radiogroup" aria-label="Pilih rating">
              {[1, 2, 3, 4, 5].map((i) => (
                <button
                  key={i}
                  type="button"
                  role="radio"
                  aria-checked={reviewForm.data.rating === i}
                  aria-label={`${i} bintang`}
                  onClick={() => reviewForm.setData("rating", i)}
                  className="p-0.5"
                >
                  <Star
                    className={`h-6 w-6 transition ${
                      i <= reviewForm.data.rating ? "fill-amber-400 text-amber-400" : "text-zinc-300 hover:text-amber-300 dark:text-zinc-700"
                    }`}
                  />
                </button>
              ))}
            </div>
            {reviewForm.errors.rating && <p className="mt-1 text-xs text-red-600">{reviewForm.errors.rating}</p>}
            <textarea
              rows={3}
              value={reviewForm.data.body}
              onChange={(e) => reviewForm.setData("body", e.target.value)}
              placeholder="Bagikan pengalaman Anda memakai produk ini (opsional)…"
              className="mt-3 w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            />
            <button
              type="submit"
              disabled={reviewForm.processing || !can_review}
              title={!can_review ? "Hanya untuk pembeli yang belum mengulas produk ini" : undefined}
              className="mt-3 rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {reviewForm.processing ? "Mengirim…" : can_review ? "Kirim ulasan" : "Ulasan hanya untuk pembeli"}
            </button>
          </form>
        )}

        <div className="mt-6 space-y-4">
          {reviews.map((rv, i) => (
            <article key={i} className="rounded-xl border border-zinc-200 p-4 dark:border-zinc-800">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-sm font-semibold">{rv.name}</span>
                <span className="text-xs text-zinc-500">{rv.created_at}</span>
              </div>
              <div className="mt-1">
                <Bintang n={rv.rating} className="h-3.5 w-3.5" />
              </div>
              {rv.body && <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">{rv.body}</p>}
            </article>
          ))}
          {reviews.length === 0 && (
            <p className="text-sm text-zinc-500">
              Belum ada ulasan.{auth_user ? "" : " Masuk dan beli produk ini untuk menjadi yang pertama mengulas."}
            </p>
          )}
        </div>
      </section>

      {related.length > 0 && (
        <section className="mt-12">
          <h2 className="mb-4 font-semibold">Produk terkait</h2>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {related.map((r) => (
              <Link key={r.id} href={`/produk/${r.slug}`} className="group rounded-xl border border-zinc-200 bg-white p-4 transition hover:border-brand-400 dark:border-zinc-800 dark:bg-zinc-900">
                <ProductThumb product={r} />
                <h3 className="mt-2 line-clamp-1 text-sm font-semibold group-hover:text-brand-700 dark:group-hover:text-brand-300">{r.title}</h3>
                <p className="text-sm font-bold text-brand-700 dark:text-brand-300">{rupiah(r.price)}</p>
              </Link>
            ))}
          </div>
        </section>
      )}
    </section>
    </>
  );
}

ProductDetail.layout = (page: ReactNode) => <GuestLayout>{page}</GuestLayout>;
