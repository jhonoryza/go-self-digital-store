import { router, usePage, useForm } from "@inertiajs/react";
import { LogOut, MailCheck } from "lucide-react";
import type { ReactNode } from "react";
import AuthLayout from "../../components/AuthLayout";
import { type SharedProps } from "../../types";

interface Props extends SharedProps {
  email: string;
  sent_otp?: string;
  resend_in?: number;
}

// Gerbang verifikasi email: masukkan OTP 6 digit yang dikirim saat daftar.
// Selama belum terverifikasi, area dashboard dan checkout tidak bisa diakses.
export default function VerifyEmail({ email, resend_in = 0 }: Props) {
  const { flash } = usePage<SharedProps>().props;
  const form = useForm({ code: "" });

  return (
    <AuthLayout title="Verifikasi Email">
      {flash?.err && (
        <div className="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-950 dark:text-red-400">{flash.err}</div>
      )}
      <div className="mb-6 text-center">
        <MailCheck className="mx-auto mb-3 h-10 w-10 text-brand-600 dark:text-brand-400" aria-hidden="true" />
        <p className="text-sm text-zinc-600 dark:text-zinc-400">
          Kode 6 digit dikirim ke <span className="font-semibold">{email}</span>.
          {Number(resend_in) > 0 && (
            <span className="block text-xs text-zinc-500">Kirim ulang tersedia dalam {resend_in} detik.</span>
          )}
        </p>
      </div>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          form.post("/verify-email");
        }}
        className="space-y-4"
      >
        <div>
          <label htmlFor="code" className="mb-1 block text-sm font-medium">
            Kode verifikasi
          </label>
          <input
            id="code"
            name="code"
            inputMode="numeric"
            autoComplete="one-time-code"
            maxLength={6}
            required
            value={form.data.code}
            onChange={(e) => form.setData("code", e.target.value.replace(/\D/g, ""))}
            className="w-full rounded-lg border-zinc-300 text-center font-mono text-2xl tracking-[0.5em] dark:border-zinc-700 dark:bg-zinc-800"
          />
          {form.errors.code && <p className="mt-1 text-sm text-red-600">{form.errors.code}</p>}
        </div>
        <button
          type="submit"
          disabled={form.processing}
          className="w-full rounded-lg bg-brand-600 px-4 py-2.5 font-semibold text-white hover:bg-brand-700 disabled:opacity-50"
        >
          {form.processing ? "Memeriksa…" : "Verifikasi"}
        </button>
      </form>
      <div className="mt-6 flex items-center justify-between text-sm">
        <button
          onClick={() => router.post("/verify-email/resend")}
          className="text-brand-600 hover:underline dark:text-brand-400"
        >
          Kirim ulang OTP
        </button>
        <button
          onClick={() => router.post("/logout")}
          className="inline-flex items-center gap-1 text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
        >
          <LogOut className="h-3.5 w-3.5" />
          Keluar
        </button>
      </div>
    </AuthLayout>
  );
}
