import { router } from "@inertiajs/react";
import { useEffect, useRef, useState } from "react";

// useInfiniteList mengakumulasi prop daftar antar halaman untuk infinite
// scroll (server tetap ber-paginasi ?page=N). Aturan sinkron:
//   - sig (signature filter) berubah  -> reset ke props terbaru
//   - page == prev.page + 1           -> append (hasil scroll)
//   - page lain (back/forward/tautan) -> reset ke props terbaru
// Sentinel ref ditempel di elemen kosong dasar daftar; IntersectionObserver
// memicu partial reload ?page=N+1 dengan hanya prop yang diberikan.
export function useInfiniteList<T>(
  itemsProp: T[],
  page: number,
  lastPage: number,
  sig: string,
  only: string[],
) {
  const [loaded, setLoaded] = useState({ sig, page, items: itemsProp });
  const [busy, setBusy] = useState(false);
  const loadingRef = useRef(false);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const onlyKey = only.join(",");

  useEffect(() => {
    setLoaded((prev) => {
      if (prev.sig !== sig) return { sig, page, items: itemsProp };
      if (page === prev.page + 1) return { ...prev, page, items: [...prev.items, ...itemsProp] };
      if (page !== prev.page) return { sig, page, items: itemsProp };
      return prev;
    });
  }, [sig, page, itemsProp]);

  useEffect(() => {
    const el = sentinelRef.current;
    if (!el) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (!entries.some((e) => e.isIntersecting) || loadingRef.current) return;
        if (loaded.page >= lastPage) return;
        loadingRef.current = true;
        setBusy(true);
        router.reload({
          only: onlyKey.split(","),
          data: { page: loaded.page + 1 },
          onFinish: () => {
            loadingRef.current = false;
            setBusy(false);
          },
        });
      },
      { rootMargin: "600px 0px" }, // mulai memuat sebelum benar-benar mentok bawah
    );
    io.observe(el);
    return () => io.disconnect();
  }, [loaded.page, lastPage, onlyKey]);

  return { items: loaded.items, busy, sentinelRef };
}
