import { translationRequest } from "./api";
import type { Resource } from "./api";

export interface BulkProgress {
  resource: string;
  checked: number;
  jobs: number;
}

// Snapshot IDs once per resource, then keep every write within the server's
// configured batch limit. Generate retains its normal eligibility/dedup checks.
export async function generateResources(
  resources: Resource[],
  onProgress: (progress: BulkProgress) => void,
  signal: AbortSignal,
) {
  let checked = 0;
  let jobs = 0;
  for (const resource of resources) {
    signal.throwIfAborted();
    onProgress({ resource: resource.label, checked, jobs });
    const selection = await translationRequest<{
      ids: number[];
      batchSize: number;
    }>(`/${resource.id}/selection`, undefined, "GET", signal);
    if (
      !Number.isInteger(selection.batchSize) ||
      selection.batchSize < 1 ||
      selection.batchSize > 50
    )
      throw new Error("Invalid translation batch limit");
    for (
      let offset = 0;
      offset < selection.ids.length;
      offset += selection.batchSize
    ) {
      signal.throwIfAborted();
      const ids = selection.ids.slice(offset, offset + selection.batchSize);
      const queued = await translationRequest<unknown[]>(
        `/${resource.id}/generate`,
        { ids, regenerate: false },
        "POST",
        signal,
      );
      checked += ids.length;
      jobs += queued.length;
      onProgress({ resource: resource.label, checked, jobs });
    }
  }
  return { checked, jobs };
}
