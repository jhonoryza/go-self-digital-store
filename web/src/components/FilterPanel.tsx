import { ChevronDown, SlidersHorizontal } from "lucide-react";
import { useState, type ReactNode } from "react";

// FilterPanel toolbar filter yang bisa disembunyikan: tombol dengan ikon
// slider + badge jumlah filter aktif; panel terbuka otomatis saat ada
// filter aktif agar state tidak tersembunyi. Dipakai katalog publik dan
// konsol admin (produk/users/orders/tiket).
export default function FilterPanel({
  activeCount = 0,
  id,
  children,
}: {
  activeCount?: number;
  id: string;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(activeCount > 0);
  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        aria-expanded={open}
        aria-controls={id}
        className="inline-flex items-center gap-2 rounded-lg border border-zinc-300 bg-white px-4 py-2 text-sm font-semibold text-zinc-700 hover:border-brand-400 hover:text-brand-700 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:border-brand-500 dark:hover:text-brand-300"
      >
        <SlidersHorizontal className="h-4 w-4" aria-hidden="true" />
        Filter
        {activeCount > 0 && (
          <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-brand-600 px-1.5 text-[10px] font-bold text-white">
            {activeCount}
          </span>
        )}
        <ChevronDown className={`h-4 w-4 text-zinc-400 transition-transform ${open ? "rotate-180" : ""}`} aria-hidden="true" />
      </button>
      {open && (
        <div id={id} className="mt-3">
          {children}
        </div>
      )}
    </div>
  );
}
