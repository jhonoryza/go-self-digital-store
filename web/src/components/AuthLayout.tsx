import { Link, usePage } from "@inertiajs/react";
import type { ReactNode } from "react";
import ThemeToggle from "./ThemeToggle";
import type { SharedProps } from "../types";

// Layout kartu terpusat untuk halaman auth.
export default function AuthLayout({ title, children }: { title: string; children: ReactNode }) {
  const { flash, app_name } = usePage<SharedProps>().props;
  return (
    <div className="relative flex min-h-dvh items-center justify-center bg-zinc-50 px-4 text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100">
      <div className="absolute right-4 top-4">
        <ThemeToggle />
      </div>
      <div className="w-full max-w-md">
        <Link href="/" className="mb-8 block text-center text-xl font-bold tracking-tight">
          {app_name ?? "Toko Digital"}
        </Link>
        {flash?.ok && (
          <div className="mb-4 rounded-lg bg-brand-50 px-4 py-3 text-sm text-brand-800 dark:bg-brand-950 dark:text-brand-300">
            {flash.ok}
          </div>
        )}
        {flash?.err && (
          <div className="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-950 dark:text-red-400">
            {flash.err}
          </div>
        )}
        <div className="rounded-xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
          <h1 className="mb-6 text-xl font-bold">{title}</h1>
          {children}
        </div>
      </div>
    </div>
  );
}
