import { Link, router, usePage } from "@inertiajs/react";
import { LayoutDashboard, LifeBuoy, LogOut, ReceiptText, Store, UserRound } from "lucide-react";
import { type ReactNode } from "react";
import BottomNav from "./BottomNav";
import FlashToast from "./FlashToast";
import ThemeToggle from "./ThemeToggle";
import type { SharedProps } from "../types";

// Layout area akun (user): header + nav tab (desktop) + bottom nav (mobile).
// Logout via POST (CSRF otomatis dari interceptor api.ts).
export default function AccountLayout({ children }: { children: ReactNode }) {
  const { auth_user, app_name, brand_logo } = usePage<SharedProps>().props;
  const { url } = usePage();
  const path = url.split("?")[0]; // cocokkan pathname saja

  const tabs = [
    ["/dashboard", "Dashboard"],
    ["/pesanan", "Pesanan"],
    ["/tiket", "Tiket"],
    ["/profil", "Profil"],
  ] as const;

  // tab aktif: sama persis atau halaman detail di bawahnya (/pesanan/{no})
  const aktif = (href: string) => path === href || path.startsWith(href + "/");

  return (
    <div className="flex min-h-dvh flex-col bg-zinc-50 pb-16 text-zinc-900 sm:pb-0 dark:bg-zinc-950 dark:text-zinc-100">
      <FlashToast />
      <header className="sticky top-0 z-40 border-b border-zinc-200 bg-white/90 backdrop-blur print:hidden dark:border-zinc-800 dark:bg-zinc-950/90">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
          <Link href="/" className="flex items-center gap-2 text-lg font-bold tracking-tight">
            {brand_logo && <img src={brand_logo} alt="" className="h-7 w-7 object-contain" />}
            {app_name ?? "Toko Digital"}
          </Link>
          <div className="flex items-center gap-1 text-sm sm:gap-3">
            <span className="hidden max-w-48 truncate text-zinc-500 sm:block">{auth_user?.email}</span>
            <ThemeToggle />
            {auth_user?.role === "admin" && (
              <Link href="/admin" className="hidden rounded-lg border border-zinc-300 px-3 py-1.5 hover:bg-zinc-100 sm:block dark:border-zinc-700 dark:hover:bg-zinc-800">
                Admin
              </Link>
            )}
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
        <nav className="mx-auto hidden max-w-6xl gap-1 overflow-x-auto px-4 sm:flex">
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
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6 sm:py-8">{children}</main>
      <BottomNav
        items={[
          { href: "/", label: "Toko", Icon: Store, active: path === "/" },
          { href: "/dashboard", label: "Dashboard", Icon: LayoutDashboard, active: path === "/dashboard" },
          { href: "/pesanan", label: "Pesanan", Icon: ReceiptText, active: aktif("/pesanan") },
          { href: "/tiket", label: "Tiket", Icon: LifeBuoy, active: aktif("/tiket") },
          { href: "/profil", label: "Profil", Icon: UserRound, active: aktif("/profil") },
        ]}
      />
    </div>
  );
}
