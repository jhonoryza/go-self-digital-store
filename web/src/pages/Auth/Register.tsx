import { useForm, Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import AuthLayout from "../../components/AuthLayout";

export default function Register() {
  const form = useForm({ name: "", email: "", password: "", password_confirmation: "" });
  return (
    <AuthLayout title="Daftar Akun">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          form.post("/register");
        }}
        className="space-y-4"
      >
        <div>
          <label htmlFor="name" className="mb-1 block text-sm font-medium">Nama</label>
          <input
            id="name"
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.name}
            autoComplete="name"
            onChange={(e) => form.setData("name", e.target.value)}
            required
            autoFocus
          />
          {form.errors.name && <p className="mt-1 text-sm text-red-600">{form.errors.name}</p>}
        </div>
        <div>
          <label htmlFor="email" className="mb-1 block text-sm font-medium">Email</label>
          <input
            id="email"
            type="email"
            className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            value={form.data.email}
            autoComplete="email"
            onChange={(e) => form.setData("email", e.target.value)}
            required
          />
          {form.errors.email && <p className="mt-1 text-sm text-red-600">{form.errors.email}</p>}
        </div>
        <div>
          <label htmlFor="password" className="mb-1 block text-sm font-medium">Password</label>
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
        <button
          type="submit"
          disabled={form.processing}
          className="w-full rounded-lg bg-brand-600 px-4 py-2.5 font-semibold text-white hover:bg-brand-700 disabled:opacity-50"
        >
          {form.processing ? "Memproses…" : "Daftar"}
        </button>
      </form>
      <p className="mt-6 text-center text-sm text-zinc-500">
        Sudah punya akun?{" "}
        <Link href="/login" className="font-semibold text-brand-600 hover:underline dark:text-brand-400">
          Masuk
        </Link>
      </p>
    </AuthLayout>
  );
}
