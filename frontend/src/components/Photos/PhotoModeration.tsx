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
      .catch(() => setMessage("Kunde inte läsa bilderna."));
  }, [id, status]);
  useEffect(() => {
    void httpClient(`${globalApi()}/photoexports?event_id=${id}`)
      .then((result) => setExports(result.json as PhotoExport[]))
      .catch(() => setMessage("Kunde inte läsa exportjobb."));
  }, [id]);

  const startExport = async () => {
    try {
      await httpClient(`${globalApi()}/photoevents/${id}/exports`, {
        method: "POST",
      });
      setMessage("Exporten har köats. Uppdatera status om en stund.");
      await refreshExports();
    } catch {
      setMessage("Kunde inte starta exporten.");
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
          ? `${counts.failed} bilder kunde inte behandlas.`
          : `${counts.succeeded} bilder behandlade.`,
      );
    } catch {
      setMessage("Modereringen misslyckades.");
    }
    setBusy(false);
    await refresh();
  };

  return (
    <main className="p-6">
      <h1 className="text-2xl">Moderering – event {id}</h1>
      <section>
        <h2>ZIP-export</h2>
        <button type="button" onClick={() => void startExport()}>
          Skapa ZIP av godkända bilder
        </button>
        <button type="button" onClick={() => void refreshExports()}>
          Uppdatera exportstatus
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
                  Ladda ned
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
          <option value="pending">Väntar</option>
          <option value="approved">Godkända</option>
          <option value="rejected">Avvisade</option>
        </select>
      </label>
      <button type="button" onClick={() => void refresh()}>
        Uppdatera
      </button>
      <p role="status">{message}</p>
      <div className="flex gap-3">
        <button
          type="button"
          disabled={busy || !selected.length}
          onClick={() => void moderate(selected, "approve")}
        >
          Godkänn valda
        </button>
        <button
          type="button"
          disabled={busy || !selected.length}
          onClick={() => void moderate(selected, "reject")}
        >
          Avvisa valda
        </button>
        <button
          type="button"
          disabled={busy || !selected.length}
          onClick={() => {
            if (window.confirm("Radera valda bilder permanent?"))
              void moderate(selected, "delete");
          }}
        >
          Radera valda
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
                alt={`Bild ${photo.id}`}
                className="h-40 w-full object-contain"
              />
            ) : (
              <div className="h-40">Borttagen</div>
            )}
            <span>{new Date(photo.uploaded_at).toLocaleString("sv-SE")}</span>
          </label>
        ))}
      </div>
      {hasMore && (
        <button
          type="button"
          onClick={() =>
            void loadMore().catch(() =>
              setMessage("Kunde inte läsa fler bilder."),
            )
          }
        >
          Ladda fler bilder
        </button>
      )}
    </main>
  );
}
