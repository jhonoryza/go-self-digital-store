import type { ReactNode } from "react";
import AdminLayout from "../../components/AdminLayout";
import type { SharedProps } from "../../types";

interface Props extends SharedProps {
  users: { id: number; email: string; name: string; role: string; verified: boolean; created_at: string }[];
}

export default function AdminUsers({ users }: Props) {
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Pengguna</h1>
      <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <table className="w-full text-sm">
          <thead className="bg-zinc-50 text-left text-zinc-500 dark:bg-zinc-950">
            <tr>
              <th className="px-4 py-2.5 font-medium">Email</th>
              <th className="px-4 py-2.5 font-medium">Nama</th>
              <th className="px-4 py-2.5 font-medium">Role</th>
              <th className="px-4 py-2.5 font-medium">Email</th>
              <th className="px-4 py-2.5 font-medium">Terdaftar</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">
            {(users ?? []).map((u) => (
              <tr key={u.id}>
                <td className="px-4 py-2.5 font-medium">{u.email}</td>
                <td className="px-4 py-2.5">{u.name}</td>
                <td className="px-4 py-2.5">
                  <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${u.role === "admin" ? "bg-zinc-900 text-white dark:bg-white dark:text-zinc-900" : "bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400"}`}>
                    {u.role}
                  </span>
                </td>
                <td className="px-4 py-2.5 text-zinc-500">{u.created_at}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

AdminUsers.layout = (page: ReactNode) => <AdminLayout>{page}</AdminLayout>;
