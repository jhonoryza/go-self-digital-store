import { router, usePage } from "@inertiajs/react";
import { Link2, LogOut, MailCheck, Wand2 } from "lucide-react";
import type { FormEvent, ReactNode } from "react";
import { useState } from "react";
import AuthLayout from "../../components/AuthLayout";
import type { SharedProps } from "../../types";

interface Props extends SharedProps {
  email?: string;
  sent_to?: string; // terisi setelah email masuk-tanpa-password dikirim
}

// Login tanpa password - dua metode yang dikirim sekaligus ke email:
//   1. Magic link: klik tautan di email untuk langsung masuk.
//   2. Kode OTP 6 digit: diketik di halaman ini.
// Keduanya sekali pakai, berlaku 15 menit, dan otomatis memverifikasi email.
export default function MagicLink() {
  const { flash, auth_user } = usePage<SharedProps>().props;
  const { sent_to } = usePage<Props>().props as unknown as Props;
  const sentEmail = sent_to || "";
  const [target, setTarget] = useState("");
  const [code, setCode] = useState("");

  const requestCode = (e: FormEvent) => {
    e.preventDefault();
    router.post("/login/link", { email: target });
  };
  const verifyCode = (e: FormEvent) => {
    e.preventDefault();
    router.post("/login/link/verify", { email: sentEmail, code });
  };

  return (
    <AuthLayout title="Masuk tanpa password">
      {flash?.err && (
        <div className="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-950 dark:text-red-400">{flash.err}</div>
      )}
      {flash?.ok && (
        <div className="mb-4 rounded-lg bg-brand-50 px-4 py-3 text-sm text-brand-800 dark:bg-brand-950 dark:text-brand-300">{flash.ok}</div>
      )}

      {!sentEmail ? (
        <form onSubmit={requestCode} className="space-y-4">
          <div>
            <label htmlFor="email" className="mb-1 block text-sm font-medium">
              Email akun
            </label>
            <input
              id="email"
              type="email"
              required
              value={target}
              onChange={(e) => setTarget(e.target.value)}
              className="w-full rounded-lg border-zinc-300 dark:border-zinc-700 dark:bg-zinc-800"
            />
            <p className="mt-1 text-xs text-zinc-500">
              Dua cara masuk dikirim sekaligus: tautan ajaib dan kode OTP 6 digit.
            </p>
          </div>
          <button
            type="submit"
            className="inline-flex w-full items-center justify-center gap-2 rounded-lg bg-brand-600 px-4 py-2.5 font-semibold text-white hover:bg-brand-700"
          >
            <Wand2 className="h-4 w-4" />
            Kirim tautan &amp; kode
          </button>
        </form>
      ) : (
        <div className="space-y-5">
          <div className="text-center">
            <MailCheck className="mx-auto mb-3 h-10 w-10 text-brand-600 dark:text-brand-400" aria-hidden="true" />
            <p className="text-sm text-zinc-600 dark:text-zinc-400">
              Tautan dan kode dikirim ke <span className="font-semibold">{sentEmail}</span>. Pilih salah satu cara:
            </p>
          </div>

          {/* Metode 1: magic link */}
          <div className="rounded-xl border border-zinc-200 p-4 dark:border-zinc-800">
            <p className="mb-1 flex items-center gap-2 text-sm font-semibold">
              <Link2 className="h-4 w-4 text-brand-600 dark:text-brand-400" /> Metode 1 - Magic link
            </p>
            <p className="text-xs text-zinc-500">
              Buka email dan klik tombol "Masuk sekarang" untuk langsung masuk di perangkat ini.
            </p>
          </div>

          {/* Metode 2: kode OTP */}
          <div className="rounded-xl border border-zinc-200 p-4 dark:border-zinc-800">
            <p className="mb-2 flex items-center gap-2 text-sm font-semibold">
              <Wand2 className="h-4 w-4 text-brand-600 dark:text-brand-400" /> Metode 2 - Kode OTP
            </p>
            <form onSubmit={verifyCode} className="space-y-3">
              <input
                id="code"
                inputMode="numeric"
                maxLength={6}
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                aria-label="Kode OTP"
                className="w-full rounded-lg border-zinc-300 text-center font-mono text-2xl tracking-[0.5em] dark:border-zinc-700 dark:bg-zinc-800"
              />
              <button
                type="submit"
                className="w-full rounded-lg bg-brand-600 px-4 py-2.5 font-semibold text-white hover:bg-brand-700"
              >
                Masuk dengan kode
              </button>
            </form>
          </div>

          <div className="flex items-center justify-between text-sm">
            <button onClick={requestCode} className="text-brand-600 hover:underline dark:text-brand-400">
              Kirim ulang email
            </button>
            {auth_user && (
              <button
                onClick={() => router.post("/logout")}
                className="inline-flex items-center gap-1 text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
              >
                <LogOut className="h-3.5 w-3.5" />
                Keluar
              </button>
            )}
          </div>
        </div>
      )}
    </AuthLayout>
  );
}
