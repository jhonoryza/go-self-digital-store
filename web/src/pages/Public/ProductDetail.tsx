import { Head, Link, router, usePage } from "@inertiajs/react";
import { Check, ShoppingCart } from "lucide-react";
import { useState, type ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import ProductTypeIcon from "../../components/ProductTypeIcon";
import { addToCart } from "../../lib/cart";
import { rupiah, type Product, type ProductImage, type SharedProps } from "../../types";

interface Props extends SharedProps {
  product: Product;
  images: ProductImage[];
  seo?: { "seo.product.title"?: string; "seo.product.description"?: string };
}

export default function ProductDetail({ product, images, seo }: Props) {
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
        <p className="mt-4 whitespace-pre-wrap leading-relaxed text-zinc-600 dark:text-zinc-400">
          {product.description}
        </p>
        <div className="mt-8 flex flex-wrap items-center justify-between gap-4 border-t border-zinc-200 pt-6 dark:border-zinc-800">
          <span className="text-2xl font-extrabold text-brand-700 dark:text-brand-300">{rupiah(product.price)}</span>
          <div className="flex items-center gap-3">
            <button
              onClick={add}
              className="inline-flex items-center gap-1.5 rounded-lg border border-brand-600 px-4 py-2.5 text-sm font-semibold text-brand-700 hover:bg-brand-50 dark:text-brand-400 dark:hover:bg-brand-950 dark:text-brand-300"
            >
              {added ? <Check className="h-4 w-4" /> : <ShoppingCart className="h-4 w-4" />}
              {added ? "Ditambahkan" : "Keranjang"}
            </button>
            <button
              onClick={() => {
                addToCart(product);
                router.visit("/checkout");
              }}
              className="rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-700"
            >
              Beli Sekarang
            </button>
          </div>
        </div>
      </div>
    </section>
    </>
  );
}

ProductDetail.layout = (page: ReactNode) => <GuestLayout>{page}</GuestLayout>;
