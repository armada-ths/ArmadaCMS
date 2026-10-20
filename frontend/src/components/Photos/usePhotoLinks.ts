import { useEffect, useRef, type Dispatch, type SetStateAction } from "react";
export type PhotoLinks = {
  id: number;
  url?: string;
  thumbnail_url?: string;
  urls_expires_at?: string;
};
export type RenewedPhotoLinks = {
  urls?: Record<string, string>;
  thumbnail_urls: Record<string, string>;
  urls_expires_at: string;
};
// Only the viewport/buffer and selected photo are eligible for renewal.
export function usePhotoLinks<T extends PhotoLinks>(
  scope: string,
  photos: T[],
  setPhotos: Dispatch<SetStateAction<T[]>>,
  activeIDs: number[],
  renew: (ids: number[], signal: AbortSignal) => Promise<RenewedPhotoLinks>,
) {
  const latest = useRef({ photos, activeIDs });
  const check = useRef<(() => void) | null>(null);
  useEffect(() => {
    latest.current = { photos, activeIDs };
  }, [photos, activeIDs]);
  useEffect(() => {
    const controller = new AbortController();
    let busy = false;
    const attempts = new Map<number, number>();
    const tick = async () => {
      if (busy || document.visibilityState !== "visible") return;
      const now = Date.now();
      const active = new Set(latest.current.activeIDs);
      const ids = latest.current.photos
        .filter(
          (photo) =>
            active.has(photo.id) &&
            (photo.thumbnail_url || photo.url) &&
            Date.parse(photo.urls_expires_at ?? "") < now + 120_000 &&
            now - (attempts.get(photo.id) ?? 0) > 60_000,
        )
        .map((photo) => photo.id);
      if (!ids.length) return;
      busy = true;
      try {
        for (let start = 0; start < ids.length; start += 100) {
          const batch = ids.slice(start, start + 100);
          batch.forEach((id) => attempts.set(id, now));
          const result = await renew(batch, controller.signal);
          if (controller.signal.aborted) return;
          setPhotos((previous) =>
            previous.map((photo) =>
              batch.includes(photo.id) &&
              result.thumbnail_urls[String(photo.id)]
                ? {
                    ...photo,
                    url: result.urls?.[String(photo.id)] ?? photo.url,
                    thumbnail_url: result.thumbnail_urls[String(photo.id)],
                    urls_expires_at: result.urls_expires_at,
                  }
                : photo,
            ),
          );
        }
      } catch {
        // Retain links; retry at most once per minute.
      } finally {
        busy = false;
      }
    };
    check.current = () => {
      void tick();
    };
    void tick();
    const interval = window.setInterval(() => {
      void tick();
    }, 30_000);
    const visible = () => {
      void tick();
    };
    document.addEventListener("visibilitychange", visible);
    return () => {
      controller.abort();
      check.current = null;
      window.clearInterval(interval);
      document.removeEventListener("visibilitychange", visible);
    };
  }, [scope, renew, setPhotos]);
  const activeKey = activeIDs.join(",");
  useEffect(() => {
    check.current?.();
  }, [activeKey]);
}
