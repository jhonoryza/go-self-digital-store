import { RotateCcw, Search } from "lucide-react";
import type { ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import { useInfiniteList } from "../../lib/infinite";
import type { SharedProps } from "../../types";

interface UserRow {
  id: number;
  email: string;
  name: string;
  role: string;
  verified: boolean;
  created_at: string;
}

interface Filters {
  q?: string;
  role?: string;
  verified?: string;
  sort?: string;
}

interface Props extends SharedProps {
  users?: UserRow[];
  page: number;
  last_page: number;
  total: number;
  filters?: Filters;
}

const SORTS: Record<string, string> = {
  terbaru: "Terbaru",
  terlama: "Terlama",
  nama: "Nama A-Z",
  email: "Email A-Z",
};

// Konsol pengguna: pencarian email/nama, filter role & verifikasi, urutan,
// dan infinite scroll (server ber-paginasi 12/halaman).
export default function AdminUsers({ users: rawUsers, page, last_page, total, filters = {} }: Props) {
  const users = rawUsers ?? [];
  const adaFilter = Boolean(filters.q || filters.role || filters.verified);
  const { items, busy, sentinelRef } = useInfiniteList(
    users,
    page,
    last_page,
    JSON.stringify(filters),
    ["users", "page", "last_page", "total"],
  );
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Pengguna</h1>

      {/* toolbar cari + filter + urut */}
      <form
        method="get"
        action="/admin/users"
        className="rounded-xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900"
      >
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div className="lg:col-span-2">
            <label htmlFor="q" className="mb-1 block text-xs font-medium text-zinc-500">
              Cari email / nama
            </label>
            <input
              id="q"
              name="q"
              type="search"
              defaultValue={filters.q || ""}
              placeholder="Cari pengguna…"
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            />
          </div>
          <div>
            <label htmlFor="role" className="mb-1 block text-xs font-medium text-zinc-500">
              Role
            </label>
            <select
              id="role"
              name="role"
              defaultValue={filters.role || ""}
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            >
              <option value="">Semua</option>
              <option value="admin">Admin</option>
              <option value="user">User</option>
            </select>
          </div>
          <div>
            <label htmlFor="verified" className="mb-1 block text-xs font-medium text-zinc-500">
              Verifikasi email
            </label>
            <select
              id="verified"
              name="verified"
              defaultValue={filters.verified || ""}
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            >
              <option value="">Semua</option>
              <option value="sudah">Terverifikasi</option>
              <option value="belum">Belum</option>
            </select>
          </div>
          <div>
            <label htmlFor="sort" className="mb-1 block text-xs font-medium text-zinc-500">
              Urutkan
            </label>
            <select
              id="sort"
              name="sort"
              defaultValue={filters.sort || "terbaru"}
              className="w-full rounded-lg border-zinc-300 text-sm dark:border-zinc-700 dark:bg-zinc-800"
            >
              {Object.entries(SORTS).map(([v, label]) => (
                <option key={v} value={v}>
                  {label}
                </option>
              ))}
            </select>
          </div>
        </div>
        <div className="mt-3 flex items-center gap-3">
          <button
            type="submit"
            className="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700"
          >
            <Search className="h-4 w-4" aria-hidden="true" /> Terapkan
          </button>
          {adaFilter && (
            <a
              href="/admin/users"
              className="inline-flex items-center gap-1.5 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
            >
              <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" /> Reset
            </a>
          )}
          <span className="ml-auto text-sm text-zinc-500">{total} pengguna</span>
        </div>
      </form>

      <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <table className="w-full text-sm">
          <thead className="bg-zinc-50 text-left text-zinc-500 dark:bg-zinc-950">
            <tr>
              <th className="px-4 py-2.5 font-medium">Email</th>
              <th className="px-4 py-2.5 font-medium">Nama</th>
              <th className="px-4 py-2.5 font-medium">Role</th>
              <th className="px-4 py-2.5 font-medium">Verifikasi</th>
              <th className="px-4 py-2.5 font-medium">Terdaftar</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">
            {items.map((u) => (
              <tr key={u.id}>
                <td className="px-4 py-2.5 font-medium">{u.email}</td>
                <td className="px-4 py-2.5">{u.name}</td>
                <td className="px-4 py-2.5">
                  <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${u.role === "admin" ? "bg-zinc-900 text-white dark:bg-white dark:text-zinc-900" : "bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400"}`}>
                    {u.role}
                  </span>
                </td>
                <td className="px-4 py-2.5">
                  <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${
                    u.verified
                      ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400"
                      : "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-400"
                  }`}>
                    {u.verified ? "terverifikasi" : "belum"}
                  </span>
                </td>
                <td className="px-4 py-2.5 text-zinc-500">{u.created_at}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {items.length === 0 && (
          <p className="p-8 text-center text-sm text-zinc-500">
            {adaFilter ? "Tidak ada pengguna yang cocok dengan filter." : "Belum ada pengguna terdaftar."}
          </p>
        )}
        <div ref={sentinelRef} aria-hidden="true" className="h-1" />
        {busy && (
          <p className="flex items-center justify-center gap-2 py-3 text-sm text-zinc-500">
            <Search className="h-4 w-4 animate-pulse" aria-hidden="true" /> Memuat pengguna lain…
          </p>
        )}
      </div>
    </div>
  );
}

AdminUsers.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
