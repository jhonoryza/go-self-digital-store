import { useForm, Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import AuthLayout from "../../components/AuthLayout";

export default function Login() {
  const form = useForm({ email: "", password: "" });
  return (
    <AuthLayout title="Masuk">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          form.post("/login");
        }}
        className="space-y-4"
      >
        <div>
          <label htmlFor="email" className="mb-1 block text-sm font-medium">Email</label>
          <input
            id="email"
            type="email"
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.email}
            onChange={(e) => form.setData("email", e.target.value)}
            required
            autoFocus
          />
          {form.errors.email && <p className="mt-1 text-sm text-red-600">{form.errors.email}</p>}
        </div>
        <div>
          <div className="mb-1 flex items-center justify-between">
            <label htmlFor="password" className="block text-sm font-medium">Password</label>
            <Link href="/lupa-password" className="text-xs text-brand-600 hover:underline dark:text-brand-400">
              Lupa password?
            </Link>
          </div>
          <input
            id="password"
            type="password"
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.password}
            onChange={(e) => form.setData("password", e.target.value)}
            required
          />
          {form.errors.password && <p className="mt-1 text-sm text-red-600">{form.errors.password}</p>}
        </div>
        <button
          type="submit"
          disabled={form.processing}
          className="w-full rounded-lg bg-brand-600 px-4 py-2.5 font-semibold text-white hover:bg-brand-700 disabled:opacity-50"
        >
          {form.processing ? "Memproses…" : "Masuk"}
        </button>
      </form>
      <p className="mt-6 text-center text-sm text-zinc-500">
        Belum punya akun?{" "}
        <Link href="/register" className="font-semibold text-brand-600 hover:underline dark:text-brand-400">
          Daftar
        </Link>
      </p>
      <p className="mt-2 text-center text-sm">
        <Link href="/login/link" className="text-zinc-500 underline decoration-dashed hover:text-zinc-800 dark:hover:text-zinc-200">
          Masuk tanpa password (tautan email)
        </Link>
      </p>
    </AuthLayout>
  );
}
