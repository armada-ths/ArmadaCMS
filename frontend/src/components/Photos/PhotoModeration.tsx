import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
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

  const moderate = async (
    ids: number[],
    action: "approve" | "reject" | "delete",
  ) => {
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
    <main className="p-6">
      <h1 className="text-2xl">Moderation – event {id}</h1>
      <section>
        <h2>ZIP export</h2>
        <button type="button" onClick={() => void startExport()}>
          Create ZIP of approved photos
        </button>
        <button type="button" onClick={() => void refreshExports()}>
          Refresh export status
        </button>
        <ul>
          {exports.map((item) => (
            <li key={item.id}>
              Export {item.id}: {item.status}{" "}
              {item.status === "completed" && (
                <button
                  type="button"
                  onClick={() => void downloadExport(item.id)}
                >
                  Download
                </button>
              )}
            </li>
          ))}
        </ul>
      </section>
      <label>
        Status:{" "}
        <select
          value={status}
          onChange={(event) => setStatus(event.target.value)}
        >
          <option value="pending">Pending</option>
          <option value="approved">Approved</option>
          <option value="rejected">Rejected</option>
        </select>
      </label>
      <button type="button" onClick={() => void refresh()}>
        Refresh
      </button>
      <p role="status">{message}</p>
      <div className="flex gap-3">
        <button
          type="button"
          disabled={busy || !selected.length}
          onClick={() => void moderate(selected, "approve")}
        >
          Approve selected
        </button>
        <button
          type="button"
          disabled={busy || !selected.length}
          onClick={() => void moderate(selected, "reject")}
        >
          Reject selected
        </button>
        <button
          type="button"
          disabled={busy || !selected.length}
          onClick={() => {
            if (window.confirm("Permanently delete the selected photos?"))
              void moderate(selected, "delete");
          }}
        >
          Delete selected
        </button>
      </div>
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4 lg:grid-cols-6">
        {photos.map((photo) => (
          <label key={photo.id} className="rounded border p-2">
            <input
              type="checkbox"
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
              <img
                src={photo.thumbnail_url}
                alt={`Photo ${photo.id}`}
                className="h-40 w-full object-contain"
              />
            ) : (
              <div className="h-40">Deleted</div>
            )}
            <span>{new Date(photo.uploaded_at).toLocaleString("en-GB")}</span>
          </label>
        ))}
      </div>
      {hasMore && (
        <button
          type="button"
          onClick={() =>
            void loadMore().catch(() =>
              setMessage("Could not load more photos."),
            )
          }
        >
          Load more photos
        </button>
      )}
    </main>
  );
}
