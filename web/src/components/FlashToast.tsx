import { usePage } from "@inertiajs/react";
import { useEffect, useState } from "react";
import type { SharedProps } from "../types";

const TUNGGAL_MS = 2000; // lama tampil sebelum mulai memudar
const FADE_MS = 300;

// Toast flash dari props.flash (isi ?ok=/&err= hasil redirect server) -
// hilang otomatis setelah 2 detik dengan fade singkat. Dipakai ketiga
// shell (publik, akun, admin). Banner inline di halaman form bukan bagian
// komponen ini dan tetap tampil menetap.
export default function FlashToast() {
  const { flash } = usePage<SharedProps>().props;
  const [msg, setMsg] = useState<{ kind: "ok" | "err"; text: string } | null>(null);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (flash?.ok) setMsg({ kind: "ok", text: flash.ok });
    else if (flash?.err) setMsg({ kind: "err", text: flash.err });
    else return;
    setVisible(true);
    const fade = setTimeout(() => setVisible(false), TUNGGAL_MS);
    const hilang = setTimeout(() => setMsg(null), TUNGGAL_MS + FADE_MS);
    return () => {
      clearTimeout(fade);
      clearTimeout(hilang);
    };
  }, [flash?.ok, flash?.err]);

  if (!msg) return null;
  return (
    <div
      role={msg.kind === "err" ? "alert" : "status"}
      className={`fixed top-4 left-1/2 z-50 -translate-x-1/2 rounded-lg px-4 py-2.5 text-sm font-medium shadow-lg transition-opacity duration-300 ${
        msg.kind === "ok" ? "bg-brand-600 text-white" : "bg-red-600 text-white"
      } ${visible ? "opacity-100" : "opacity-0"}`}
    >
      {msg.text}
    </div>
  );
}
