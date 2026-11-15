import { useForm } from "@inertiajs/react";
import { LogIn } from "lucide-react";
import GuestLayout from "../../components/GuestLayout";

// Dev login-as: masukkan user_id akun mana pun untuk masuk tanpa password.
// Sengaja TIDAK mengenumerasi daftar akun (temuan audit) - halaman dev tetap
// tidak bocor daftar user bila flag tersalahkan aktif di host lain.
// Hanya tersedia saat server dijalankan dengan STORE_DEV_LOGIN=1.
export default function DevLoginAs() {
  const form = useForm({ user_id: "" });

  return (
    <GuestLayout>
      <section className="mx-auto max-w-md px-4 py-12">
        <p className="mb-1 text-xs font-semibold uppercase tracking-widest text-brass-600 dark:text-brass-400">
          Alat development
        </p>
        <h1 className="text-2xl font-bold tracking-tight">Masuk sebagai akun</h1>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          Buat sesi langsung untuk{" "}
          <code className="rounded bg-zinc-100 px-1.5 py-0.5 font-mono text-xs dark:bg-zinc-800">
            user_id
          </code>{" "}
          mana pun, tanpa password. Aktif hanya saat server berjalan dengan
          <code className="mx-1 rounded bg-zinc-100 px-1.5 py-0.5 font-mono text-xs dark:bg-zinc-800">
            STORE_DEV_LOGIN=1
          </code>
          - route ini tidak terdaftar di produksi.
        </p>

        <form
          onSubmit={(e) => {
            e.preventDefault();
            form.post("/dev/login");
          }}
          className="mt-6 space-y-4"
        >
          <div>
            <label htmlFor="user_id" className="mb-1 block text-sm font-medium">
              User ID
            </label>
            <input
              id="user_id"
              type="number"
              min="1"
              required
              value={form.data.user_id}
              onChange={(e) => form.setData("user_id", e.target.value)}
              className="w-full rounded-lg border-zinc-300 font-mono dark:border-zinc-700 dark:bg-zinc-800"
            />
            {form.errors.user_id && <p className="mt-1 text-sm text-red-600">{form.errors.user_id}</p>}
          </div>
          <button
            type="submit"
            disabled={form.processing}
            className="inline-flex w-full items-center justify-center gap-2 rounded-lg bg-brand-600 px-4 py-2.5 font-semibold text-white hover:bg-brand-700 disabled:opacity-50"
          >
            <LogIn className="h-4 w-4" />
            Masuk sebagai
          </button>
        </form>
      </section>
    </GuestLayout>
  );
}
