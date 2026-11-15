import { Link, router, usePage } from "@inertiajs/react";
import { type ReactNode } from "react";
import ThemeToggle from "./ThemeToggle";
import type { SharedProps } from "../types";

// Layout konsol admin: header + tab nav.
export default function AdminLayout({ children }: { children: ReactNode }) {
  const { auth_user, app_name, brand_logo, flash } = usePage<SharedProps>().props;
  const { url } = usePage();

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

  return (
    <div className="flex min-h-screen flex-col bg-zinc-100 text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100">
      {flash?.ok && (
        <div className="fixed top-4 left-1/2 z-50 -translate-x-1/2 rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white shadow-lg" role="status">
          {flash.ok}
        </div>
      )}
      {flash?.err && (
        <div className="fixed top-4 left-1/2 z-50 -translate-x-1/2 rounded-lg bg-red-600 px-4 py-2.5 text-sm font-medium text-white shadow-lg" role="status">
          {flash.err}
        </div>
      )}
      <header className="border-b border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4">
          <div className="flex items-center gap-3">
            <span className="rounded-md bg-zinc-900 px-2 py-1 text-xs font-bold text-white dark:bg-white dark:text-zinc-900">ADMIN</span>
            <Link href="/" className="flex items-center gap-2 font-bold">
              {brand_logo && <img src={brand_logo} alt="" className="h-6 w-6 object-contain" />}
              {app_name ?? "Toko Digital"}
            </Link>
          </div>
          <div className="flex items-center gap-3 text-sm">
            <span className="text-zinc-500">{auth_user?.email}</span>
            <ThemeToggle />
            <Link href="/dashboard" className="rounded-lg border border-zinc-300 px-3 py-1.5 hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800">
              Saya
            </Link>
            <button
              onClick={() => router.post("/logout")}
              className="rounded-lg px-3 py-1.5 text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-800"
            >
              Keluar
            </button>
          </div>
        </div>
        <nav className="mx-auto flex max-w-7xl gap-1 overflow-x-auto px-4">
          {tabs.map(([href, label]) => (
            <Link
              key={href}
              href={href}
              className={`whitespace-nowrap border-b-2 px-3 py-3 text-sm font-medium ${
                url === href
                  ? "border-brand-600 text-brand-700 dark:text-brand-400"
                  : "border-transparent text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
              }`}
            >
              {label}
            </Link>
          ))}
        </nav>
      </header>
      <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-8">{children}</main>
    </div>
  );
}
