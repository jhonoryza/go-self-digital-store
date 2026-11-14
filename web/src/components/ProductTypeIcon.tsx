import { KeyRound, Link2, Package } from "lucide-react";

// Ikon jenis produk digital (file/key/link) - dipakai lintas halaman
// katalog dan keranjang agar switch type → ikon tidak diduplikasi.
export default function ProductTypeIcon({ type, className }: { type: string; className?: string }) {
  if (type === "key") return <KeyRound className={className ?? "h-6 w-6"} />;
  if (type === "link") return <Link2 className={className ?? "h-6 w-6"} />;
  return <Package className={className ?? "h-6 w-6"} />;
}
