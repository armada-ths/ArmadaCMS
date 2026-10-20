import { useCallback, useEffect, useRef, useState } from "react";
import { useParams } from "react-router-dom";
import { useGetOne } from "react-admin";
import {
  Alert,
  Box,
  Button,
  FormControl,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Typography,
} from "@mui/material";
import { PhotoModerationGrid } from "./PhotoModerationGrid";
import { canSelectPhoto, type Photo } from "./photoModerationTypes";
import { usePhotoLinks, type RenewedPhotoLinks } from "./usePhotoLinks";
import { httpClient } from "../../dataProvider";
import globalApi from "../../context/globalApi";
import { parsePositiveIntegerId } from "../../utils/parsePositiveIntegerId";

type PhotoExport = { id: number; status: string; error?: string };

export function PhotoModeration() {
  const { id } = useParams();
  const eventId = parsePositiveIntegerId(id);
  if (eventId === null)
    return <Alert severity="error">Invalid photo event ID.</Alert>;
  return <PhotoModerationContent key={eventId} eventId={eventId} />;
}

function PhotoModerationContent({ eventId }: { eventId: number }) {
  const { data: event } = useGetOne<{ id: number; name: string }>(
    "photoevents",
    { id: eventId ?? 0 },
    { enabled: eventId !== null },
  );
  const [photos, setPhotos] = useState<Photo[]>([]);
  const [selected, setSelected] = useState<number[]>([]);
  const [status, setStatus] = useState("pending");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [exports, setExports] = useState<PhotoExport[]>([]);
  const [visibleIDs, setVisibleIDs] = useState<number[]>([]);
  const [loadingPhotos, setLoadingPhotos] = useState(false);
  const requestGeneration = useRef(0);
  const loadingRef = useRef(false);
  const [page, setPage] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [now, setNow] = useState(Date.now);
  const selectedIds = photos
    .filter(
      (photo) =>
        photo.status === status &&
        selected.includes(photo.id) &&
        canSelectPhoto(photo, now),
    )
    .map((photo) => photo.id);

  useEffect(() => {
    if (status !== "rejected") return;
    const timer = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => window.clearInterval(timer);
  }, [status]);

  const refresh = useCallback(async () => {
    if (eventId === null) return;
    const generation = ++requestGeneration.current;
    loadingRef.current = true;
    setLoadingPhotos(true);
    setSelected([]);
    try {
      const result = await httpClient(
        `${globalApi()}/eventphotos?event_id=${eventId}&status=${status}`,
      );
      if (generation !== requestGeneration.current) return;
      setPhotos(result.json as Photo[]);
      setPage(0);
      setHasMore((result.json as Photo[]).length === 200);
    } catch {
      if (generation === requestGeneration.current)
        setMessage("Could not load the photos.");
    } finally {
      if (generation === requestGeneration.current) {
        loadingRef.current = false;
        setLoadingPhotos(false);
      }
    }
  }, [eventId, status]);

  const loadMore = async () => {
    if (loadingRef.current || eventId === null) return;
    const generation = requestGeneration.current;
    const nextPage = page + 1;
    loadingRef.current = true;
    setLoadingPhotos(true);
    try {
      const result = await httpClient(
        `${globalApi()}/eventphotos?event_id=${eventId}&status=${status}&page=${nextPage}`,
      );
      if (generation !== requestGeneration.current) return;
      const incoming = result.json as Photo[];
      setPhotos((current) => [
        ...current,
        ...incoming.filter(
          (photo) => !current.some((existing) => existing.id === photo.id),
        ),
      ]);
      setPage(nextPage);
      setHasMore(incoming.length === 200);
    } catch {
      if (generation === requestGeneration.current)
        setMessage("Could not load more photos.");
    } finally {
      if (generation === requestGeneration.current) {
        loadingRef.current = false;
        setLoadingPhotos(false);
      }
    }
  };

  const renewLinks = useCallback(
    async (ids: number[], signal: AbortSignal) => {
      const result = await httpClient(`${globalApi()}/eventphotos/refresh`, {
        method: "POST",
        body: JSON.stringify({ event_id: eventId, ids }),
        signal,
      });
      return result.json as RenewedPhotoLinks;
    },
    [eventId],
  );
  usePhotoLinks(
    `${eventId}:${status}`,
    photos,
    setPhotos,
    visibleIDs,
    renewLinks,
  );

  const refreshExports = useCallback(async () => {
    if (eventId === null) return;
    const result = await httpClient(
      `${globalApi()}/photoexports?event_id=${eventId}`,
    );
    setExports(result.json as PhotoExport[]);
  }, [eventId]);

  useEffect(() => {
    let cancelled = false;
    const requests = requestGeneration;
    void Promise.resolve().then(() => {
      if (!cancelled) return refresh();
    });
    return () => {
      cancelled = true;
      requests.current++;
    };
  }, [refresh]);
  useEffect(() => {
    if (eventId === null) return;
    void httpClient(`${globalApi()}/photoexports?event_id=${eventId}`)
      .then((result) => setExports(result.json as PhotoExport[]))
      .catch(() => setMessage("Could not load the export jobs."));
  }, [eventId]);

  const startExport = async () => {
    try {
      await httpClient(`${globalApi()}/photoevents/${eventId}/exports`, {
        method: "POST",
      });
      setMessage("The export is queued. Refresh its status in a moment.");
      await refreshExports();
    } catch {
      setMessage("Could not start the export.");
    }
  };

  const downloadExport = async (exportID: number) => {
    const result = await httpClient(`${globalApi()}/photoexports/${exportID}`);
    const url = (result.json as { download_url?: string }).download_url;
    if (url) window.location.assign(url);
  };

  const moderate = async (ids: number[], action: "approve" | "reject") => {
    const generation = requestGeneration.current;
    setBusy(true);
    setMessage("");
    try {
      const result = await httpClient(`${globalApi()}/eventphotos/batch`, {
        method: "POST",
        body: JSON.stringify({ ids, action }),
      });
      const counts = result.json as { succeeded: number; failed: number };
      setMessage(
        counts.failed
          ? `${counts.failed} photos could not be processed.`
          : `${counts.succeeded} photos processed.`,
      );
    } catch {
      setMessage("Moderation failed.");
    }
    setBusy(false);
    if (generation === requestGeneration.current) await refresh();
  };

  if (eventId === null) {
    return <Alert severity="error">Invalid photo event ID.</Alert>;
  }

  return (
    <Box component="main" sx={{ p: 3 }}>
      <Typography variant="h5" gutterBottom>
        Guest photos{event?.name ? ` · ${event.name}` : ""}
      </Typography>
      <Paper variant="outlined" sx={{ p: 2, mb: 3 }}>
        <Typography variant="h6" gutterBottom>
          ZIP export
        </Typography>
        <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1.5 }}>
          <Button variant="outlined" onClick={() => void startExport()}>
            Create ZIP of approved photos
          </Button>
          <Button variant="outlined" onClick={() => void refreshExports()}>
            Refresh export status
          </Button>
        </Box>
        {exports.map((item) => (
          <Stack
            key={item.id}
            direction="row"
            alignItems="center"
            gap={1}
            sx={{ mt: 1 }}
          >
            <Typography variant="body2">
              Export {item.id}: {item.status}
            </Typography>
            {item.status === "completed" && (
              <Button size="small" onClick={() => void downloadExport(item.id)}>
                Download
              </Button>
            )}
          </Stack>
        ))}
      </Paper>
      <Box
        sx={{
          display: "flex",
          alignItems: "center",
          flexWrap: "wrap",
          gap: 1.5,
          mb: 2,
        }}
      >
        <FormControl size="small" sx={{ width: 180, flexShrink: 0 }}>
          <InputLabel id="photo-status-label">Status</InputLabel>
          <Select
            labelId="photo-status-label"
            label="Status"
            value={status}
            onChange={(event) => {
              requestGeneration.current++;
              setPhotos([]);
              setHasMore(false);
              setStatus(event.target.value);
              setSelected([]);
              setNow(Date.now());
            }}
          >
            <MenuItem value="pending">Pending</MenuItem>
            <MenuItem value="approved">Approved</MenuItem>
            <MenuItem value="rejected">Rejected</MenuItem>
          </Select>
        </FormControl>
        <Button
          variant="outlined"
          disabled={loadingPhotos || busy}
          onClick={() => void refresh()}
        >
          Refresh
        </Button>
        <Button
          disabled={
            busy ||
            loadingPhotos ||
            status === "approved" ||
            !selectedIds.length
          }
          onClick={() => void moderate(selectedIds, "approve")}
        >
          Approve selected
        </Button>
        <Button
          disabled={
            busy ||
            loadingPhotos ||
            status === "rejected" ||
            !selectedIds.length
          }
          onClick={() => void moderate(selectedIds, "reject")}
        >
          Reject selected
        </Button>
      </Box>
      {status === "rejected" && (
        <Alert severity="info" sx={{ mb: 2 }}>
          Rejected photos are hidden from the gallery immediately. They can be
          approved again within 24 hours, after which they will be permanently
          deleted. Deleted photos cannot be restored.
        </Alert>
      )}
      {message && (
        <Alert
          severity={
            message.startsWith("Could not") || message === "Moderation failed."
              ? "error"
              : "info"
          }
          sx={{ mb: 2 }}
        >
          {message}
        </Alert>
      )}
      <PhotoModerationGrid
        photos={photos}
        selected={selected}
        busy={busy}
        now={now}
        onToggle={(id, checked) =>
          setSelected((current) =>
            checked
              ? [...new Set([...current, id])]
              : current.filter((value) => value !== id),
          )
        }
        onRangeChange={setVisibleIDs}
      />

      {hasMore && (
        <Button
          sx={{ mt: 2 }}
          disabled={loadingPhotos || busy}
          onClick={() =>
            void loadMore().catch(() =>
              setMessage("Could not load more photos."),
            )
          }
        >
          Load more photos
        </Button>
      )}
    </Box>
  );
}
