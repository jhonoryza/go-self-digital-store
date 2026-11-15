// Bootstrap Inertia: shell statis fetch page JSON → createInertiaApp dengan
// page awal; komponen halaman di-resolve via import.meta.glob; CsrfSync menyinkronkan
// token CSRF props → header X-CSRF-Token pada semua visit tulis.

import { createInertiaApp } from "@inertiajs/react";
import { usePage } from "@inertiajs/react";
import { useEffect, type ComponentType, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { fetchInitialPage, setCsrf } from "./api";
// Ubuntu (Google Fonts) di-host sendiri via @fontsource — tanpa <link> eksternal,
// font-display: swap bawaan fontsource.
import "@fontsource/ubuntu/400.css";
import "@fontsource/ubuntu/500.css";
import "@fontsource/ubuntu/700.css";
import "@fontsource/ubuntu-mono/400.css";
import "./css/app.css";
import type { Page, PageProps, SharedPageProps } from "@inertiajs/core";
import type { SharedProps } from "./types";

const pageFiles = import.meta.glob("./pages/**/*.tsx");

// Komponen halaman Inertia - boleh menyematkan layout persisten (.layout).
type PageComponent = ComponentType<any> & { layout?: (p: ReactNode) => ReactNode };

async function resolve(name: string, _page?: Page<SharedPageProps>): Promise<PageComponent> {
  const mod = (await pageFiles[`./pages/${name}.tsx`]()) as { default: PageComponent };
  if (!mod.default.layout) {
    mod.default.layout = (p) => p; // tanpa layout khusus
  }
  return mod.default;
}

// Sinkronkan tema toko pilihan admin ke <html data-theme> (shell statis CF
// Pages tidak bisa menyuntik atribut ini dari server).
function ThemeSync() {
  const { props } = usePage<SharedProps>();
  useEffect(() => {
    document.documentElement.dataset.theme = props.theme ?? "";
  }, [props.theme]);
  return null;
}

// Sinkronkan token CSRF dari props halaman (berubah tiap render).
function CsrfSync() {
  const { props } = usePage<SharedProps>();
  useEffect(() => {
    setCsrf(props.csrf_token ?? "");
  }, [props.csrf_token]);
  return null;
}

const initialPage = await fetchInitialPage(window.location.pathname + window.location.search);

createInertiaApp<SharedProps>({
  page: initialPage as Page<PageProps & SharedPageProps>,
  resolve,
  setup({ el, App, props }) {
    createRoot(el).render(
      <App {...props}>
        {({ Component, props: pageProps }) => {
          // layout persisten diapply manual (callback children menonaktifkan
          // penerapan layout bawaan App). resolve() selalu menyetel layout
          // berupa fungsi - perketat tipe union bawaan Inertia (yang juga
          // mendukung komponen/array layout) ke bentuk yang dipakai app ini.
          const Page = Component as PageComponent;
          const layout = Page.layout ?? ((p: ReactNode) => p);
          return (
            <>
              <CsrfSync />
              <ThemeSync />
              {layout(<Page {...pageProps} />)}
            </>
          );
        }}
      </App>,
    );
  },
});
