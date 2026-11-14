import { Link, usePage } from "@inertiajs/react";
import { useEffect, useState, type ReactNode } from "react";
import type { SharedProps } from "../types";

// Flash toast dari ?ok=/&err= — hilang otomatis.
function Flash() {
  const { flash } = usePage<SharedProps>().props;
  const [msg, setMsg] = useState<{ kind: "ok" | "err"; text: string } | null>(null);
  useEffect(() => {
    if (flash?.ok) setMsg({ kind: "ok", text: flash.ok });
    else if (flash?.err) setMsg({ kind: "err", text: flash.err });
    else setMsg(null);
    if (flash?.ok || flash?.err) {
      const t = setTimeout(() => setMsg(null), 4000);
      return () => clearTimeout(t);
    }
  }, [flash?.ok, flash?.err]);
  if (!msg) return null;
  return (
    <div
      className={`fixed top-4 left-1/2 z-50 -translate-x-1/2 rounded-lg px-4 py-2.5 text-sm font-medium shadow-lg ${
        msg.kind === "ok" ? "bg-brand-600 text-white" : "bg-red-600 text-white"
      }`}
      role="status"
    >
      {msg.text}
    </div>
  );
}

export default function GuestLayout({ children }: { children: ReactNode }) {
  const { auth_user, app_name } = usePage<SharedProps>().props;
  const [open, setOpen] = useState(false);
  return (
    <div className="flex min-h-screen flex-col bg-zinc-50 text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100">
      <Flash />
      <header className="sticky top-0 z-40 border-b border-zinc-200 bg-white/90 backdrop-blur dark:border-zinc-800 dark:bg-zinc-950/90">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
          <Link href="/" className="text-lg font-bold tracking-tight">
            {app_name ?? "Toko Digital"}
          </Link>
          <nav className="hidden items-center gap-6 text-sm font-medium sm:flex">
            <Link href="/produk" className="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white">
              Produk
            </Link>
            <Link href="/ketentuan" className="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white">
              Ketentuan
            </Link>
            {auth_user ? (
              <>
                {auth_user.role === "admin" && (
                  <Link href="/admin" className="text-brand-600 hover:text-brand-800 dark:text-brand-400">
                    Admin
                  </Link>
                )}
                <Link href="/dashboard" className="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white">
                  Dashboard
                </Link>
              </>
            ) : (
              <>
                <Link href="/login" className="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white">
                  Masuk
                </Link>
                <Link
                  href="/register"
                  className="rounded-lg bg-brand-600 px-3.5 py-2 text-white hover:bg-brand-700"
                >
                  Daftar
                </Link>
              </>
            )}
          </nav>
          <button
            className="sm:hidden rounded-md p-2 text-zinc-600 dark:text-zinc-300"
            onClick={() => setOpen(!open)}
            aria-label="Buka menu"
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              {open ? <path d="M18 6 6 18M6 6l12 12" /> : <path d="M4 7h16M4 12h16M4 17h16" />}
            </svg>
          </button>
        </div>
        {open && (
          <nav className="border-t border-zinc-200 px-4 py-3 text-sm sm:hidden dark:border-zinc-800">
            <Link href="/produk" className="block py-2" onClick={() => setOpen(false)}>Produk</Link>
            <Link href="/ketentuan" className="block py-2" onClick={() => setOpen(false)}>Ketentuan</Link>
            {auth_user ? (
              <Link href="/dashboard" className="block py-2" onClick={() => setOpen(false)}>Dashboard</Link>
            ) : (
              <>
                <Link href="/login" className="block py-2" onClick={() => setOpen(false)}>Masuk</Link>
                <Link href="/register" className="block py-2 font-semibold text-brand-600" onClick={() => setOpen(false)}>
                  Daftar
                </Link>
              </>
            )}
          </nav>
        )}
      </header>
      <main className="flex-1">{children}</main>
      <footer className="border-t border-zinc-200 py-8 dark:border-zinc-800">
        <div className="mx-auto flex max-w-6xl flex-col gap-2 px-4 text-sm text-zinc-500 sm:flex-row sm:items-center sm:justify-between">
          <span>© {new Date().getFullYear()} {app_name ?? "Toko Digital"} — produk digital</span>
          <div className="flex gap-4">
            <Link href="/privacy" className="hover:text-zinc-800 dark:hover:text-zinc-200">Privasi</Link>
            <Link href="/ketentuan" className="hover:text-zinc-800 dark:hover:text-zinc-200">Ketentuan</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
