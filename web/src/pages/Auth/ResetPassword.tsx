import { useForm, Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import AuthLayout from "../../components/AuthLayout";
import type { SharedProps } from "../../types";

interface Props extends SharedProps {
  token: string;
}

export default function ResetPassword({ token }: Props) {
  const form = useForm({ token, password: "", password_confirmation: "" });
  return (
    <AuthLayout title="Password Baru">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          form.post("/reset-password");
        }}
        className="space-y-4"
      >
        <div>
          <label htmlFor="password" className="mb-1 block text-sm font-medium">Password baru</label>
          <input
            id="password"
            type="password"
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.password}
            onChange={(e) => form.setData("password", e.target.value)}
            required
            autoFocus
          />
          {form.errors.password && <p className="mt-1 text-sm text-red-600">{form.errors.password}</p>}
        </div>
        <div>
          <label htmlFor="password_confirmation" className="mb-1 block text-sm font-medium">Ulangi password</label>
          <input
            id="password_confirmation"
            type="password"
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.password_confirmation}
            onChange={(e) => form.setData("password_confirmation", e.target.value)}
            required
          />
          {form.errors.password_confirmation && (
            <p className="mt-1 text-sm text-red-600">{form.errors.password_confirmation}</p>
          )}
        </div>
        {form.errors.token && (
          <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-950 dark:text-red-400">
            {form.errors.token}{" "}
            <Link href="/lupa-password" className="underline">Minta tautan baru</Link>
          </div>
        )}
        <button
          type="submit"
          disabled={form.processing}
          className="w-full rounded-lg bg-brand-600 px-4 py-2.5 font-semibold text-white hover:bg-brand-700 disabled:opacity-50"
        >
          {form.processing ? "Menyimpan…" : "Simpan password baru"}
        </button>
      </form>
    </AuthLayout>
  );
}
