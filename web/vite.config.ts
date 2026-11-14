import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// Produksi: dist/ dipindah ke Cloudflare Pages (shell statis + aset ber-hash).
// Dev: vite di :5173 dengan proxy ke Go di :8080 — semua path halaman/API
// diteruskan ke backend, vite tetap melayani /src, /@vite, /node_modules.
// Ini mencerminkan produksi split-origin (CF Pages → API VPS) tanpa CORS lokal.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      "^/((?!src|node_modules|@vite|@react-refresh|index\\.html|assets).*)": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  build: {
    manifest: true,
    outDir: "dist",
  },
});
