import { useForm, Link } from "@inertiajs/react";
import type { ReactNode } from "react";
import AuthLayout from "../../components/AuthLayout";

export default function ForgotPassword() {
  const form = useForm({ email: "" });
  return (
    <AuthLayout title="Lupa Password">
      <p className="mb-4 text-sm text-zinc-600 dark:text-zinc-400">
        Masukkan email akun Anda — tautan reset akan dikirim bila email terdaftar.
      </p>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          form.post("/lupa-password");
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
        <button
          type="submit"
          disabled={form.processing}
          className="w-full rounded-lg bg-brand-600 px-4 py-2.5 font-semibold text-white hover:bg-brand-700 disabled:opacity-50"
        >
          {form.processing ? "Mengirim…" : "Kirim tautan reset"}
        </button>
      </form>
      <p className="mt-6 text-center text-sm text-zinc-500">
        <Link href="/login" className="font-semibold text-brand-600 hover:underline dark:text-brand-400">
          ← Kembali ke masuk
        </Link>
      </p>
    </AuthLayout>
  );
}
