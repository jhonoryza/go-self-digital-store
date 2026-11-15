import { Link } from "@inertiajs/react";
import type { LucideIcon } from "lucide-react";

export interface BottomNavItem {
  href: string;
  label: string;
  Icon: LucideIcon;
  active?: boolean;
  badge?: number;
  onClick?: () => void; // tanpa navigasi (mis. buka menu lain)
}

// BottomNav bar navigasi bawah ala aplikasi - hanya tampil di layar sempit
// (di bawah breakpoint sm). Item aktif diwarnai brand; badge opsional
// (jumlah keranjang). Dipakai shell publik dan dashboard user/admin.
export default function BottomNav({ items }: { items: BottomNavItem[] }) {
  return (
    <nav
      aria-label="Navigasi utama"
      className="fixed inset-x-0 bottom-0 z-40 border-t border-zinc-200 bg-white/95 pb-[env(safe-area-inset-bottom)] backdrop-blur print:hidden sm:hidden dark:border-zinc-800 dark:bg-zinc-950/95"
    >
      <div className="grid" style={{ gridTemplateColumns: `repeat(${items.length}, minmax(0, 1fr))` }}>
        {items.map(({ href, label, Icon, active = false, badge = 0, onClick }) => {
          const cls = `flex flex-col items-center gap-0.5 py-2.5 text-[11px] font-medium ${
            active
              ? "text-brand-600 dark:text-brand-400"
              : "text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
          }`;
          const inner = (
            <>
              <span className="relative">
                <Icon className="h-5 w-5" aria-hidden="true" />
                {badge > 0 && (
                  <span className="absolute -top-1.5 -right-2 flex h-4 min-w-4 items-center justify-center rounded-full bg-brand-600 px-1 text-[9px] font-bold text-white">
                    {badge}
                  </span>
                )}
              </span>
              {label}
            </>
          );
          return onClick ? (
            <button key={href} type="button" onClick={onClick} className={cls} aria-expanded={undefined}>
              {inner}
            </button>
          ) : (
            <Link key={href} href={href} aria-current={active ? "page" : undefined} className={cls}>
              {inner}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
