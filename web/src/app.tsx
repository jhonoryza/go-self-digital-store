// Bootstrap Inertia: shell statis fetch page JSON → createInertiaApp dengan
// page awal; komponen halaman di-resolve via import.meta.glob; CsrfSync menyinkronkan
// token CSRF props → header X-CSRF-Token pada semua visit tulis.

import { createInertiaApp } from "@inertiajs/react";
import { usePage } from "@inertiajs/react";
import { useEffect, type ComponentType, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { fetchInitialPage, setCsrf } from "./api";
import type { Page, PageProps, SharedPageProps } from "@inertiajs/core";
import type { SharedProps } from "./types";

const pageFiles = import.meta.glob("./pages/**/*.tsx");

type ResolvedPage = { default: ComponentType & { layout?: (p: ReactNode) => ReactNode } };

async function resolve(name: string, _page?: Page<SharedPageProps>): Promise<ResolvedPage> {
  const mod = (await pageFiles[`./pages/${name}.tsx`]()) as ResolvedPage;
  if (!mod.default.layout) {
    mod.default.layout = (p) => p; // tanpa layout khusus
  }
  return mod;
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
        {(Component, pageProps) => (
          <>
            <CsrfSync />
            <Component {...pageProps} />
          </>
        )}
      </App>,
    );
  },
});
