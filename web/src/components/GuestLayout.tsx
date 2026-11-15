import { Link, usePage } from "@inertiajs/react";
import { Home, LogIn, Menu, Package, ReceiptText, ShoppingCart, X } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";
import BottomNav from "./BottomNav";
import FlashToast from "./FlashToast";
import ThemeToggle from "./ThemeToggle";
import { cartCount, getCart } from "../lib/cart";
import type { SharedProps } from "../types";

// Jumlah item keranjang - ikut event "cart-change" dari lib/cart.
function useCartCount() {
  const [count, setCount] = useState(cartCount(getCart()));
  useEffect(() => {
    const sync = () => setCount(cartCount(getCart()));
    window.addEventListener("cart-change", sync);
    window.addEventListener("pageshow", sync);
    return () => {
      window.removeEventListener("cart-change", sync);
      window.removeEventListener("pageshow", sync);
    };
  }, []);
  return count;
}

function CartBadge() {
  const count = useCartCount();
  return (
    <Link href="/keranjang" className="relative rounded-md p-2 text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white" aria-label="Keranjang">
      <ShoppingCart className="h-5 w-5" />
      {count > 0 && (
        <span className="absolute -top-0.5 -right-0.5 flex h-5 w-5 items-center justify-center rounded-full bg-brand-600 text-[10px] font-bold text-white">
          {count}
        </span>
      )}
    </Link>
  );
}

export default function GuestLayout({ children }: { children: ReactNode }) {
  const { auth_user, app_name, brand_logo, dev_login } = usePage<SharedProps>().props;
  const { url } = usePage();
  const path = url.split("?")[0]; // pathname saja agar filter (?q=…) tak mematikan highlight
  const [open, setOpen] = useState(false);
  const count = useCartCount();

  const bottomItems = [
    { href: "/", label: "Beranda", Icon: Home, active: path === "/" },
    { href: "/produk", label: "Produk", Icon: Package, active: path === "/produk" || path.startsWith("/produk/") },
    { href: "/keranjang", label: "Keranjang", Icon: ShoppingCart, active: path === "/keranjang", badge: count },
    auth_user
      ? { href: "/pesanan", label: "Pesanan", Icon: ReceiptText, active: path === "/pesanan" || path.startsWith("/pesanan/") }
      : { href: "/login", label: "Masuk", Icon: LogIn, active: path === "/login" },
  ];

  return (
    <div className="flex min-h-screen flex-col bg-zinc-50 pb-16 text-zinc-900 sm:pb-0 dark:bg-zinc-950 dark:text-zinc-100">
      <FlashToast />
      <header className="sticky top-0 z-40 border-b border-zinc-200 bg-white/90 backdrop-blur dark:border-zinc-800 dark:bg-zinc-950/90">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4">
          <Link href="/" className="flex items-center gap-2 text-lg font-bold tracking-tight">
            {brand_logo && <img src={brand_logo} alt="" className="h-7 w-7 object-contain" />}
            {app_name ?? "Toko Digital"}
          </Link>
          <nav className="hidden items-center gap-6 text-sm font-medium sm:flex">
            <Link href="/produk" className="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white">
              Produk
            </Link>
            <Link href="/ketentuan" className="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white">
              Ketentuan
            </Link>
            {dev_login && (
              <Link href="/dev/login" className="text-brass-600 hover:text-brass-800 dark:text-brass-400">
                Dev Login
              </Link>
            )}
            <CartBadge />
            <ThemeToggle />
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
          <div className="flex items-center gap-1 sm:hidden">
            <ThemeToggle />
            <CartBadge />
            <button
              className="rounded-md p-2 text-zinc-600 dark:text-zinc-300"
              onClick={() => setOpen(!open)}
              aria-label="Buka menu"
              aria-expanded={open}
            >
              {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
          </div>
        </div>
        {open && (
          <nav className="border-t border-zinc-200 px-4 py-3 text-sm sm:hidden dark:border-zinc-800">
            {/* utama sudah di bottom nav; menu ini hanya pelengkap */}
            <Link href="/ketentuan" className="block py-2" onClick={() => setOpen(false)}>Ketentuan</Link>
            {dev_login && (
              <Link href="/dev/login" className="block py-2 text-brass-600 dark:text-brass-400" onClick={() => setOpen(false)}>
                Dev Login
              </Link>
            )}
            {auth_user ? (
              <>
                {auth_user.role === "admin" && (
                  <Link href="/admin" className="block py-2 text-brand-600 dark:text-brand-400" onClick={() => setOpen(false)}>
                    Admin
                  </Link>
                )}
                <Link href="/dashboard" className="block py-2" onClick={() => setOpen(false)}>Dashboard</Link>
              </>
            ) : (
              <Link href="/register" className="block py-2 font-semibold text-brand-600 dark:text-brand-400" onClick={() => setOpen(false)}>
                Daftar
              </Link>
            )}
          </nav>
        )}
      </header>
      <main className="flex-1">{children}</main>
      <footer className="border-t border-zinc-200 py-8 pb-24 sm:pb-8 dark:border-zinc-800">
        <div className="mx-auto flex max-w-6xl flex-col gap-2 px-4 text-sm text-zinc-500 sm:flex-row sm:items-center sm:justify-between">
          <span>© {new Date().getFullYear()} {app_name ?? "Toko Digital"} - produk digital</span>
          <div className="flex gap-4">
            <Link href="/privacy" className="hover:text-zinc-800 dark:hover:text-zinc-200">Privasi</Link>
            <Link href="/ketentuan" className="hover:text-zinc-800 dark:hover:text-zinc-200">Ketentuan</Link>
          </div>
        </div>
      </footer>
      <BottomNav items={bottomItems} />
    </div>
  );
}
