import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useWindowVirtualizer } from "@tanstack/react-virtual";
import { Box, Checkbox, Paper, Typography } from "@mui/material";
import { canSelectPhoto, type Photo } from "./photoModerationTypes";

function ModerationThumbnail({ photo }: { photo: Photo }) {
  const [failed, setFailed] = useState(false);
  if (!photo.thumbnail_url || failed)
    return (
      <Box sx={{ height: 160, display: "grid", placeItems: "center" }}>
        {!photo.thumbnail_url ? "Deleted" : "Image unavailable"}
      </Box>
    );
  return (
    <Box
      component="img"
      src={photo.thumbnail_url}
      alt={`Photo ${photo.id}`}
      loading="lazy"
      decoding="async"
      onError={() => setFailed(true)}
      sx={{ height: 160, width: "100%", objectFit: "contain" }}
    />
  );
}

export function PhotoModerationGrid({
  photos,
  selected,
  busy,
  now,
  onToggle,
  onRangeChange,
}: {
  photos: Photo[];
  selected: number[];
  busy: boolean;
  now: number;
  onToggle: (id: number, checked: boolean) => void;
  onRangeChange: (ids: number[]) => void;
}) {
  const container = useRef<HTMLDivElement>(null);
  const [layout, setLayout] = useState({ width: 0, columns: 1, top: 0 });
  const pendingFocus = useRef<number | null>(null);
  const virtualizer = useWindowVirtualizer({
    count: Math.ceil(photos.length / layout.columns),
    estimateSize: () => 320,
    overscan: 2,
    scrollMargin: layout.top,
    useFlushSync: false,
  });
  useLayoutEffect(() => {
    const node = container.current;
    if (!node) return;
    const measure = () => {
      const next = {
        width: node.clientWidth,
        columns: Math.max(1, Math.floor((node.clientWidth + 16) / 186)),
        top: node.getBoundingClientRect().top + window.scrollY,
      };
      setLayout((old) =>
        old.width === next.width &&
        old.top === next.top &&
        old.columns === next.columns
          ? old
          : next,
      );
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(node);
    // Export/status messages above the grid can change its document offset.
    if (node.parentElement) observer.observe(node.parentElement);
    window.addEventListener("resize", measure);
    return () => {
      observer.disconnect();
      window.removeEventListener("resize", measure);
    };
  }, []);
  useLayoutEffect(() => {
    virtualizer.measure();
  }, [layout.width, layout.columns, photos.length, virtualizer]);
  const rows = virtualizer.getVirtualItems();
  const first = rows[0]?.index ?? 0;
  const last = rows[rows.length - 1]?.index ?? -1;
  useEffect(() => {
    onRangeChange(
      photos
        .slice(first * layout.columns, (last + 1) * layout.columns)
        .map((photo) => photo.id),
    );
  }, [first, last, photos, layout.columns, onRangeChange]);
  useLayoutEffect(() => {
    if (pendingFocus.current === null) return;
    const input = container.current?.querySelector<HTMLInputElement>(
      `[data-photo-id="${pendingFocus.current}"] input`,
    );
    if (input) {
      input.focus({ preventScroll: true });
      pendingFocus.current = null;
    }
  });
  return (
    <Box
      ref={container}
      sx={{ position: "relative", height: virtualizer.getTotalSize() }}
      onKeyDownCapture={(event) => {
        if (
          !(event.target instanceof HTMLInputElement) ||
          event.target.type !== "checkbox"
        )
          return;
        const id = Number(
          event.target
            .closest("[data-photo-id]")
            ?.getAttribute("data-photo-id"),
        );
        const index = photos.findIndex((photo) => photo.id === id);
        const direction =
          event.key === "Tab"
            ? event.shiftKey
              ? -1
              : 1
            : event.key === "ArrowRight"
              ? 1
              : event.key === "ArrowLeft"
                ? -1
                : event.key === "ArrowDown"
                  ? layout.columns
                  : event.key === "ArrowUp"
                    ? -layout.columns
                    : 0;
        let next = index + direction;
        while (
          direction &&
          next >= 0 &&
          next < photos.length &&
          !canSelectPhoto(photos[next], now)
        )
          next += direction;
        if (!direction || next < 0 || next >= photos.length) return;
        event.preventDefault();
        pendingFocus.current = photos[next].id;
        virtualizer.scrollToIndex(Math.floor(next / layout.columns), {
          align: "auto",
        });
        const input = container.current?.querySelector<HTMLInputElement>(
          `[data-photo-id="${photos[next].id}"] input`,
        );
        if (input) {
          input.focus({ preventScroll: true });
          pendingFocus.current = null;
        }
      }}
    >
      {rows.map((row) => (
        <Box
          key={row.key}
          ref={virtualizer.measureElement}
          data-index={row.index}
          data-gallery-row={row.index}
          sx={{
            position: "absolute",
            top: 0,
            left: 0,
            width: "100%",
            pb: 2,
            transform: `translateY(${row.start - layout.top}px)`,
            display: "grid",
            gridTemplateColumns: `repeat(${layout.columns}, minmax(0, 1fr))`,
            gap: 2,
          }}
        >
          {photos
            .slice(row.index * layout.columns, (row.index + 1) * layout.columns)
            .map((photo) => (
              <Paper
                key={photo.id}
                data-photo-id={photo.id}
                component="label"
                variant="outlined"
                sx={{
                  p: 1,
                  minWidth: 0,
                  overflowWrap: "anywhere",
                  display: "flex",
                  flexDirection: "column",
                  cursor: canSelectPhoto(photo, now) ? "pointer" : "default",
                  color: canSelectPhoto(photo, now)
                    ? "inherit"
                    : "text.disabled",
                }}
              >
                <Checkbox
                  sx={{ alignSelf: "center" }}
                  disabled={busy || !canSelectPhoto(photo, now)}
                  checked={
                    canSelectPhoto(photo, now) && selected.includes(photo.id)
                  }
                  onChange={(event) => onToggle(photo.id, event.target.checked)}
                />
                <ModerationThumbnail
                  key={photo.thumbnail_url ?? "deleted"}
                  photo={photo}
                />
                <Typography variant="caption">
                  {new Date(photo.uploaded_at).toLocaleString("en-GB")}
                </Typography>
                {photo.status === "rejected" && (
                  <Typography variant="caption">
                    {canSelectPhoto(photo, now) && photo.rejected_delete_at
                      ? `Can approve until ${new Date(photo.rejected_delete_at).toLocaleString("en-GB")}`
                      : photo.thumbnail_url
                        ? "Awaiting permanent deletion"
                        : "Permanently deleted"}
                  </Typography>
                )}
                {photo.ai_review_status !== "not_scanned" && (
                  <Typography variant="caption">
                    AI:{" "}
                    {photo.ai_review_status === "error"
                      ? "unavailable — review manually"
                      : photo.ai_review_status}
                  </Typography>
                )}
                {photo.ai_review_status === "review" &&
                  photo.ai_likelihoods && (
                    <Typography variant="caption">
                      {Object.entries(photo.ai_likelihoods)
                        .filter(
                          ([, likelihood]) =>
                            !["VERY_UNLIKELY", "UNLIKELY"].includes(likelihood),
                        )
                        .map(
                          ([category, likelihood]) =>
                            `${category}: ${likelihood}`,
                        )
                        .join(", ")}
                    </Typography>
                  )}
                {photo.moderation_source === "vision_safe_search" && (
                  <Typography variant="caption">
                    Automatically approved
                  </Typography>
                )}
              </Paper>
            ))}
        </Box>
      ))}
    </Box>
  );
}
