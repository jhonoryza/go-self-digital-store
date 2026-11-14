import { useForm } from "@inertiajs/react";
import type { ReactNode } from "react";
import AccountLayout from "../../components/AccountLayout";
import type { SharedProps } from "../../types";

interface Props extends SharedProps {
  // user diambil dari shared props auth_user
}

export default function Profile({ auth_user }: Props) {
  const profile = useForm({ name: auth_user?.name ?? "", email: auth_user?.email ?? "" });
  const pass = useForm({ current_password: "", password: "", password_confirmation: "" });

  return (
    <div className="max-w-xl space-y-8">
      <h1 className="text-2xl font-bold">Profil</h1>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          profile.post("/profil");
        }}
        className="space-y-4 rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900"
      >
        <h2 className="font-semibold">Data akun</h2>
        <div>
          <label htmlFor="name" className="mb-1 block text-sm font-medium">Nama</label>
          <input
            id="name"
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={profile.data.name}
            onChange={(e) => profile.setData("name", e.target.value)}
            required
          />
          {profile.errors.name && <p className="mt-1 text-sm text-red-600">{profile.errors.name}</p>}
        </div>
        <div>
          <label htmlFor="email" className="mb-1 block text-sm font-medium">Email</label>
          <input
            id="email"
            type="email"
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={profile.data.email}
            onChange={(e) => profile.setData("email", e.target.value)}
            required
          />
          {profile.errors.email && <p className="mt-1 text-sm text-red-600">{profile.errors.email}</p>}
        </div>
        <button
          type="submit"
          disabled={profile.processing}
          className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50"
        >
          {profile.processing ? "Menyimpan…" : "Simpan profil"}
        </button>
      </form>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          pass.post("/profil/password", { onSuccess: () => pass.reset() });
        }}
        className="space-y-4 rounded-xl border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-900"
      >
        <h2 className="font-semibold">Ganti password</h2>
        <div>
          <label htmlFor="current_password" className="mb-1 block text-sm font-medium">Password saat ini</label>
          <input
            id="current_password"
            type="password"
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={pass.data.current_password}
            onChange={(e) => pass.setData("current_password", e.target.value)}
            required
          />
          {pass.errors.current_password && <p className="mt-1 text-sm text-red-600">{pass.errors.current_password}</p>}
        </div>
        <div>
          <label htmlFor="password" className="mb-1 block text-sm font-medium">Password baru</label>
          <input
            id="password"
            type="password"
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={pass.data.password}
            onChange={(e) => pass.setData("password", e.target.value)}
            required
          />
          {pass.errors.password && <p className="mt-1 text-sm text-red-600">{pass.errors.password}</p>}
        </div>
        <div>
          <label htmlFor="password_confirmation" className="mb-1 block text-sm font-medium">Ulangi password baru</label>
          <input
            id="password_confirmation"
            type="password"
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={pass.data.password_confirmation}
            onChange={(e) => pass.setData("password_confirmation", e.target.value)}
            required
          />
          {pass.errors.password_confirmation && (
            <p className="mt-1 text-sm text-red-600">{pass.errors.password_confirmation}</p>
          )}
        </div>
        <button
          type="submit"
          disabled={pass.processing}
          className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-700 disabled:opacity-50"
        >
          {pass.processing ? "Menyimpan…" : "Ganti password"}
        </button>
      </form>
    </div>
  );
}

Profile.layout = (page: ReactNode) => <AccountLayout>{page}</AccountLayout>;
