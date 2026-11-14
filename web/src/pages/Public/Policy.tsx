import type { ReactNode } from "react";
import GuestLayout from "../../components/GuestLayout";
import type { SharedProps } from "../../types";

interface Props extends SharedProps {
  kind: "privacy" | "terms";
}

const privacy = [
  ["Data yang kami kumpulkan", "Saat checkout kami mencatat email dan detail order (produk, jumlah, status pembayaran). Bila Anda membuat akun, kami menyimpan nama, email, dan hash password."],
  ["Penggunaan data", "Data dipakai untuk memproses order, mengirim produk ke email Anda, invoice, dan menangani tiket bantuan. Kami tidak menjual data ke pihak ketiga."],
  ["Pembayaran", "Pembayaran diproses oleh gateway pembayaran (Midtrans, Tripay, Xendit, dll). Kami tidak menyimpan nomor kartu atau kredensial pembayaran."],
  ["Cookie", "Kami memakai cookie sesi untuk menjaga Anda tetap masuk dan cookie fungsional lainnya. Tidak ada cookie iklan/pelacak pihak ketiga."],
  ["Retensi", "Order dan invoice disimpan selama toko beroperasi untuk kebutuhan akuntansi. Anda bisa meminta penghapusan data akun lewat tiket bantuan."],
];

const terms = [
  ["Lisensi", "Setiap pembelian memberi Anda lisensi pribadi, bukan komersial, kecuali dinyatakan lain pada deskripsi produk. Dilarang menjual ulang atau membagikan berkas secara publik."],
  ["Pengiriman", "Produk dikirim otomatis ke email setelah pembayaran terverifikasi oleh gateway. Tautan unduh juga tersedia di halaman bayar dan dashboard Anda."],
  ["Refund", "Karena sifat produk digital yang langsung terkirim, pembelian tidak dapat dibatalkan setelah produk terkirim. Bila berkas rusak/tidak bisa diakses, ajukan tiket - kami perbaiki atau ganti."],
  ["Akun", "Anda bertanggung jawab menjaga kerahasiaan akun. Satu akun per orang; aktivitas penipuan mengakibatkan penangguhan."],
  ["Perubahan", "Ketentuan dapat diperbarui sewaktu-waktu; versi terbaru berlaku sejak dipublikasikan di halaman ini."],
];

export default function Policy({ kind }: Props) {
  const sections = kind === "privacy" ? privacy : terms;
  return (
    <section className="mx-auto max-w-3xl px-4 py-12">
      <h1 className="mb-8 text-3xl font-bold">{kind === "privacy" ? "Kebijakan Privasi" : "Syarat & Ketentuan"}</h1>
      <div className="space-y-8">
        {sections.map(([title, body], i) => (
          <div key={title}>
            <h2 className="mb-2 font-semibold">
              {i + 1}. {title}
            </h2>
            <p className="leading-relaxed text-zinc-600 dark:text-zinc-400">{body}</p>
          </div>
        ))}
      </div>
    </section>
  );
}

Policy.layout = (page: ReactNode) => <GuestLayout>{page}</GuestLayout>;
