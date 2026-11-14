import { Link, router, usePage } from "@inertiajs/react";
import { type ReactNode } from "react";
import type { SharedProps } from "../types";

// Layout area akun (user): header + nav tab. Logout via POST (CSRF otomatis
// dari interceptor api.ts).
export default function AccountLayout({ children }: { children: ReactNode }) {
  const { auth_user, app_name } = usePage<SharedProps>().props;
  const { url } = usePage();

  const tabs = [
    ["/dashboard", "Dashboard"],
    ["/pesanan", "Pesanan"],
    ["/tiket", "Tiket"],
    ["/profil", "Profil"],
  ] as const;

  return (
    <div className="flex min-h-screen flex-col bg-zinc-50 text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100">
      <header className="sticky top-0 z-40 border-b border-zinc-200 bg-white/90 backdrop-blur dark:border-zinc-800 dark:bg-zinc-950/90">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
          <Link href="/" className="text-lg font-bold tracking-tight">
            {app_name ?? "Toko Digital"}
          </Link>
          <div className="flex items-center gap-3 text-sm">
            <span className="hidden text-zinc-500 sm:block">{auth_user?.email}</span>
            {auth_user?.role === "admin" && (
              <Link href="/admin" className="rounded-lg border border-zinc-300 px-3 py-1.5 hover:bg-zinc-100 dark:border-zinc-700 dark:hover:bg-zinc-800">
                Admin
              </Link>
            )}
            <button
              onClick={() => router.post("/logout")}
              className="rounded-lg px-3 py-1.5 text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-800"
            >
              Keluar
            </button>
          </div>
        </div>
        <nav className="mx-auto flex max-w-6xl gap-1 overflow-x-auto px-4">
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
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-8">{children}</main>
    </div>
  );
}
