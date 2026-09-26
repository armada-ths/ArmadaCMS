import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { useGetOne } from "react-admin";
import {
  Alert,
  Box,
  Button,
  Checkbox,
  FormControl,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Typography,
} from "@mui/material";
import { httpClient } from "../../dataProvider";
import globalApi from "../../context/globalApi";

type Photo = {
  id: number;
  status: string;
  thumbnail_url?: string;
  uploaded_at: string;
};
type PhotoExport = { id: number; status: string; error?: string };

export function PhotoModeration() {
  const { id } = useParams();
  const { data: event } = useGetOne<{ id: number; name: string }>(
    "photoevents",
    { id: id ?? "" },
    { enabled: Boolean(id) },
  );
  const [photos, setPhotos] = useState<Photo[]>([]);
  const [selected, setSelected] = useState<number[]>([]);
  const [status, setStatus] = useState("pending");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [exports, setExports] = useState<PhotoExport[]>([]);
  const [page, setPage] = useState(0);
  const [hasMore, setHasMore] = useState(false);

  const refresh = useCallback(async () => {
    const result = await httpClient(
      `${globalApi()}/eventphotos?event_id=${id}&status=${status}`,
    );
    setPhotos(result.json as Photo[]);
    setPage(0);
    setHasMore((result.json as Photo[]).length === 200);
    setSelected([]);
  }, [id, status]);

  const loadMore = async () => {
    const nextPage = page + 1;
    const result = await httpClient(
      `${globalApi()}/eventphotos?event_id=${id}&status=${status}&page=${nextPage}`,
    );
    const incoming = result.json as Photo[];
    setPhotos((current) => [...current, ...incoming]);
    setPage(nextPage);
    setHasMore(incoming.length === 200);
  };

  const refreshExports = useCallback(async () => {
    const result = await httpClient(
      `${globalApi()}/photoexports?event_id=${id}`,
    );
    setExports(result.json as PhotoExport[]);
  }, [id]);

  useEffect(() => {
    void httpClient(
      `${globalApi()}/eventphotos?event_id=${id}&status=${status}`,
    )
      .then((result) => {
        setPhotos(result.json as Photo[]);
        setSelected([]);
        setPage(0);
        setHasMore((result.json as Photo[]).length === 200);
      })
      .catch(() => setMessage("Could not load the photos."));
  }, [id, status]);
  useEffect(() => {
    void httpClient(`${globalApi()}/photoexports?event_id=${id}`)
      .then((result) => setExports(result.json as PhotoExport[]))
      .catch(() => setMessage("Could not load the export jobs."));
  }, [id]);

  const startExport = async () => {
    try {
      await httpClient(`${globalApi()}/photoevents/${id}/exports`, {
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
    await refresh();
  };

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
            onChange={(event) => setStatus(event.target.value)}
          >
            <MenuItem value="pending">Pending</MenuItem>
            <MenuItem value="approved">Approved</MenuItem>
            <MenuItem value="rejected">Rejected</MenuItem>
          </Select>
        </FormControl>
        <Button variant="outlined" onClick={() => void refresh()}>
          Refresh
        </Button>
        <Button
          disabled={busy || !selected.length}
          onClick={() => void moderate(selected, "approve")}
        >
          Approve selected
        </Button>
        <Button
          disabled={busy || !selected.length}
          onClick={() => void moderate(selected, "reject")}
        >
          Reject selected
        </Button>
      </Box>
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
      <Box
        sx={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fill, minmax(170px, 1fr))",
          gap: 2,
        }}
      >
        {photos.map((photo) => (
          <Paper
            key={photo.id}
            component="label"
            variant="outlined"
            sx={{
              p: 1,
              display: "flex",
              flexDirection: "column",
              cursor: "pointer",
            }}
          >
            <Checkbox
              checked={selected.includes(photo.id)}
              onChange={(event) =>
                setSelected((current) =>
                  event.target.checked
                    ? [...current, photo.id]
                    : current.filter((value) => value !== photo.id),
                )
              }
            />
            {photo.thumbnail_url ? (
              <Box
                component="img"
                src={photo.thumbnail_url}
                alt={`Photo ${photo.id}`}
                sx={{ height: 160, width: "100%", objectFit: "contain" }}
              />
            ) : (
              <Box sx={{ height: 160, display: "grid", placeItems: "center" }}>
                Deleted
              </Box>
            )}
            <Typography variant="caption">
              {new Date(photo.uploaded_at).toLocaleString("en-GB")}
            </Typography>
          </Paper>
        ))}
      </Box>
      {hasMore && (
        <Button
          sx={{ mt: 2 }}
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
