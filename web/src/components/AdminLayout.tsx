import { Link, router, usePage } from "@inertiajs/react";
import {
  CreditCard,
  LayoutDashboard,
  LifeBuoy,
  LogOut,
  Mail,
  MoreHorizontal,
  Package,
  ReceiptText,
  Settings,
  Users,
} from "lucide-react";
import { useState, type ReactNode } from "react";
import BottomNav from "./BottomNav";
import FlashToast from "./FlashToast";
import ThemeToggle from "./ThemeToggle";
import type { SharedProps } from "../types";

// Layout konsol admin: header + tab nav (desktop) + bottom nav (mobile,
// pelengkap lewat sheet "Menu"). Flash ditampilkan sebagai toast.
export default function AdminLayout({ children }: { children: ReactNode }) {
  const { auth_user, app_name, brand_logo } = usePage<SharedProps>().props;
  const { url } = usePage();
  const path = url.split("?")[0]; // cocokkan pathname saja - filter (?q=…) tak boleh mematikan highlight
  const [menuOpen, setMenuOpen] = useState(false);

  const tabs = [
    ["/admin", "Dashboard"],
    ["/admin/orders", "Orders"],
    ["/admin/produk", "Produk"],
    ["/admin/users", "Users"],
    ["/admin/tiket", "Tiket"],
    ["/admin/maillog", "Mail Log"],
    ["/admin/gateway", "Gateway"],
    ["/admin/setting", "Setting"],
  ] as const;

  // tab aktif: "/admin" (dashboard) hanya persis; tab lain juga mencakup
  // halaman detail di bawahnya (/admin/orders/{no})
  const aktif = (href: string) =>
    href === "/admin" ? path === "/admin" : path === href || path.startsWith(href + "/");

  const moreItems = [
    { href: "/admin/tiket", label: "Tiket", Icon: LifeBuoy },
    { href: "/admin/maillog", label: "Mail Log", Icon: Mail },
    { href: "/admin/gateway", label: "Gateway", Icon: CreditCard },
    { href: "/admin/setting", label: "Setting", Icon: Settings },
  ] as const;

  return (
    <div className="flex min-h-dvh flex-col bg-zinc-100 pb-16 text-zinc-900 sm:pb-0 dark:bg-zinc-950 dark:text-zinc-100">
      <FlashToast />
      <header className="sticky top-0 z-40 border-b border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4">
          <div className="flex items-center gap-3">
            <span className="hidden rounded-md bg-zinc-900 px-2 py-1 text-xs font-bold text-white sm:inline-block dark:bg-white dark:text-zinc-900">ADMIN</span>
            <Link href="/" className="flex items-center gap-2 font-bold">
              <img src={brand_logo || "/icons/icon-96.png"} alt="" className="h-6 w-6 rounded-md object-contain" />
              {app_name ?? "Toko Digital"}
            </Link>
          </div>
          <div className="flex items-center gap-1 text-sm sm:gap-3">
            <span className="hidden max-w-48 truncate text-zinc-500 sm:block">{auth_user?.email}</span>
            <ThemeToggle />
            <Link href="/dashboard" className="hidden rounded-lg border border-zinc-300 px-3 py-1.5 hover:bg-zinc-100 sm:block dark:border-zinc-700 dark:hover:bg-zinc-800">
              Saya
            </Link>
            <button
              onClick={() => router.post("/logout")}
              aria-label="Keluar"
              className="flex items-center gap-1.5 rounded-lg px-2 py-2 text-zinc-600 hover:bg-zinc-100 sm:px-3 dark:text-zinc-400 dark:hover:bg-zinc-800"
            >
              <LogOut className="h-5 w-5 sm:h-4 sm:w-4" aria-hidden="true" />
              <span className="hidden sm:inline">Keluar</span>
            </button>
          </div>
        </div>
        <nav className="mx-auto hidden max-w-7xl gap-1 overflow-x-auto px-4 sm:flex">
          {tabs.map(([href, label]) => (
            <Link
              key={href}
              href={href}
              className={`whitespace-nowrap border-b-2 px-3 py-3 text-sm font-medium ${
                aktif(href)
                  ? "border-brand-600 text-brand-700 dark:text-brand-400"
                  : "border-transparent text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
              }`}
            >
              {label}
            </Link>
          ))}
        </nav>
      </header>
      <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-6 sm:py-8">{children}</main>

      {/* sheet pelengkap bottom nav mobile */}
      {menuOpen && (
        <div className="fixed inset-0 z-40 sm:hidden" role="dialog" aria-label="Menu lainnya" onClick={() => setMenuOpen(false)}>
          <div className="absolute inset-0 bg-black/40" />
          <div
            className="absolute inset-x-3 bottom-20 rounded-xl border border-zinc-200 bg-white p-2 shadow-xl dark:border-zinc-800 dark:bg-zinc-900"
            onClick={(e) => e.stopPropagation()}
          >
            {moreItems.map(({ href, label, Icon }) => (
              <Link
                key={href}
                href={href}
                onClick={() => setMenuOpen(false)}
                className={`flex items-center gap-3 rounded-lg px-3 py-3 text-sm font-medium ${
                  aktif(href)
                    ? "bg-brand-50 text-brand-700 dark:bg-brand-950 dark:text-brand-300"
                    : "text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-800"
                }`}
              >
                <Icon className="h-5 w-5 text-zinc-500 dark:text-zinc-400" aria-hidden="true" />
                {label}
              </Link>
            ))}
          </div>
        </div>
      )}
      <BottomNav
        items={[
          { href: "/admin", label: "Dashboard", Icon: LayoutDashboard, active: path === "/admin" },
          { href: "/admin/orders", label: "Orders", Icon: ReceiptText, active: aktif("/admin/orders") },
          { href: "/admin/produk", label: "Produk", Icon: Package, active: aktif("/admin/produk") },
          { href: "/admin/users", label: "Users", Icon: Users, active: aktif("/admin/users") },
          {
            href: "#menu",
            label: "Menu",
            Icon: MoreHorizontal,
            active: moreItems.some(({ href }) => aktif(href)),
            onClick: () => setMenuOpen(!menuOpen),
          },
        ]}
      />
    </div>
  );
}
