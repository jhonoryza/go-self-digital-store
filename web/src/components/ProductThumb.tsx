import ProductTypeIcon from "./ProductTypeIcon";
import type { Product } from "../types";

// Kartu gambar produk: foto pertama bila tersedia (dari R2), ikon jenis
// sebagai fallback. Dipakai di landing, katalog, dan keranjang.
export default function ProductThumb({
  product,
  className = "h-28",
  iconClassName = "h-6 w-6",
}: {
  product: Pick<Product, "type" | "title" | "image_url">;
  className?: string;
  iconClassName?: string;
}) {
  if (product.image_url) {
    return (
      <div className={`mb-3 overflow-hidden rounded-lg bg-brand-50 dark:bg-brand-950 ${className}`}>
        <img
          src={product.image_url}
          alt={product.title}
          className="h-full w-full object-cover transition group-hover:scale-[1.03]"
          loading="lazy"
        />
      </div>
    );
  }
  return (
    <div className={`mb-3 flex items-center justify-center rounded-lg bg-brand-50 text-3xl dark:bg-brand-950 ${className}`}>
      <ProductTypeIcon type={product.type} className={iconClassName} />
    </div>
  );
}
