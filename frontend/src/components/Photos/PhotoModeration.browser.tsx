import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import { useState } from "react";
import { PhotoModerationGrid } from "./PhotoModerationGrid";
import { PhotoModeration } from "./PhotoModeration";
import { usePhotoLinks, type RenewedPhotoLinks } from "./usePhotoLinks";
import type { Photo } from "./photoModerationTypes";

const mocks = vi.hoisted(() => ({ http: vi.fn() }));
vi.mock("../../dataProvider", () => ({ httpClient: mocks.http }));
vi.mock("../../context/globalApi", () => ({ default: () => "/api" }));
vi.mock("react-admin", () => ({
  useGetOne: () => ({ data: { id: 1, name: "Test event" } }),
}));
vi.mock("react-router-dom", () => ({ useParams: () => ({ id: "1" }) }));

const image =
  "data:image/svg+xml," +
  encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="640" height="480"><rect width="640" height="480" fill="pink"/></svg>',
  );
const makePhotos = (count: number, start = 1, status = "pending"): Photo[] =>
  Array.from({ length: count }, (_, i) => ({
    id: start + i,
    status,
    thumbnail_url: image,
    uploaded_at: "2026-10-20T12:00:00Z",
    ai_review_status: i % 3 ? "safe" : "review",
    ai_likelihoods: { adult: "POSSIBLE", racy: "POSSIBLE" },
    urls_expires_at: "2099-01-01T00:00:00Z",
  }));
let container: HTMLDivElement, root: Root;
beforeEach(() => {
  container = document.createElement("div");
  container.style.width = "960px";
  document.body.appendChild(container);
  root = createRoot(container);
  mocks.http.mockReset();
  vi.restoreAllMocks();
  window.scrollTo(0, 0);
});
afterEach(() => {
  root.unmount();
  container.remove();
  window.scrollTo(0, 0);
});
const cards = () => container.querySelectorAll<HTMLElement>("[data-photo-id]");
test("bounds mounted cards, preserves selection after scrolling and measures tall rows", async () => {
  function Grid() {
    const [selected, setSelected] = useState<number[]>([]);
    return (
      <>
        <output>{selected.join(",")}</output>
        <PhotoModerationGrid
          photos={items}
          selected={selected}
          busy={false}
          now={Date.now()}
          onRangeChange={() => {}}
          onToggle={(id, checked) =>
            setSelected((old) =>
              checked ? [...old, id] : old.filter((value) => value !== id),
            )
          }
        />
      </>
    );
  }
  const items = makePhotos(1000);
  items[0].ai_likelihoods = Object.fromEntries(
    Array.from({ length: 25 }, (_, i) => ["Long category " + i, "POSSIBLE"]),
  );
  root.render(<Grid />);
  await vi.waitFor(() => expect(cards().length).toBeGreaterThan(0));
  expect(cards().length).toBeLessThan(50);
  await page.getByRole("checkbox").first().click();
  expect(container.querySelector("output")?.textContent).toBe("1");
  window.scrollTo(0, 10000);
  await vi.waitFor(() =>
    expect(Number(cards()[0]?.dataset.photoId)).toBeGreaterThan(20),
  );
  expect(cards().length).toBeLessThan(50);
  window.scrollTo(0, 0);
  await vi.waitFor(() =>
    expect(
      container.querySelector<HTMLInputElement>('[data-photo-id="1"] input')
        ?.checked,
    ).toBe(true),
  );
  const rows = Array.from(
    container.querySelectorAll<HTMLElement>("[data-gallery-row]"),
  );
  for (let i = 1; i < rows.length; i++)
    expect(rows[i].getBoundingClientRect().top).toBeGreaterThanOrEqual(
      rows[i - 1].getBoundingClientRect().bottom - 1,
    );
  container.style.width = "390px";
  await vi.waitFor(() =>
    expect(container.querySelector("[data-gallery-row]")?.children.length).toBe(
      2,
    ),
  );
});
test("distinguishes a broken image from a permanently deleted image", async () => {
  const items = makePhotos(2);
  items[0].thumbnail_url = "data:image/jpeg;base64,AA";
  items[1].thumbnail_url = undefined;
  root.render(
    <PhotoModerationGrid
      photos={items}
      selected={[]}
      busy={false}
      now={Date.now()}
      onRangeChange={() => {}}
      onToggle={() => {}}
    />,
  );
  await expect
    .element(page.getByText("Image unavailable", { exact: true }))
    .toBeVisible();
  await expect
    .element(page.getByText("Deleted", { exact: true }))
    .toBeVisible();
});
test("batch moderation includes selected photos outside the viewport", async () => {
  mocks.http.mockImplementation(async (input: string) => {
    if (input.includes("/batch")) return { json: { succeeded: 2, failed: 0 } };
    if (input.includes("/photoexports")) return { json: [] };
    return { json: makePhotos(200) };
  });
  root.render(<PhotoModeration />);
  await vi.waitFor(() => expect(cards().length).toBeGreaterThan(0));
  await page.getByRole("checkbox").first().click();
  window.scrollTo(0, 9000);
  await vi.waitFor(() =>
    expect(Number(cards()[0]?.dataset.photoId)).toBeGreaterThan(20),
  );
  const id = Number(cards()[0].dataset.photoId);
  container.querySelector<HTMLInputElement>('input[type="checkbox"]')!.click();
  await page
    .getByRole("button", { name: "Approve selected", exact: true })
    .click();
  const call = mocks.http.mock.calls.find(([url]) => url.includes("/batch"));
  expect(JSON.parse(call![1].body)).toEqual({
    ids: [1, id],
    action: "approve",
  });
});
test("ignores delayed pagination after changing the status filter", async () => {
  let finish: (value: unknown) => void = () => {};
  mocks.http.mockImplementation(async (input: string) => {
    if (input.includes("/photoexports")) return { json: [] };
    if (input.includes("page=1"))
      return await new Promise((resolve) => {
        finish = resolve;
      });
    return {
      json: input.includes("status=approved")
        ? makePhotos(1, 5000, "approved")
        : makePhotos(200),
    };
  });
  root.render(<PhotoModeration />);
  await vi.waitFor(() => expect(cards().length).toBeGreaterThan(0));
  await page
    .getByRole("button", { name: "Load more photos", exact: true })
    .click();
  await page.getByRole("combobox").click();
  await page.getByRole("option", { name: "Approved", exact: true }).click();
  await vi.waitFor(() => expect(cards()[0]?.dataset.photoId).toBe("5000"));
  finish({ json: makePhotos(200, 201) });
  await vi.waitFor(() => expect(cards().length).toBe(1));
  expect(container.querySelector('[data-photo-id="201"]')).toBeNull();
  await expect
    .element(
      page.getByRole("button", { name: "Approve selected", exact: true }),
    )
    .toBeDisabled();
});

test("renews only visible expired links, pauses while hidden and coalesces requests", async () => {
  let visibility: DocumentVisibilityState = "hidden";
  vi.spyOn(document, "visibilityState", "get").mockImplementation(
    () => visibility,
  );
  let finish: (value: RenewedPhotoLinks) => void = () => {};
  const renew = vi.fn(async (ids: number[], signal: AbortSignal) => {
    if (!ids.length || signal.aborted) throw new Error("Unexpected renewal");
    return await new Promise<RenewedPhotoLinks>((resolve) => {
      finish = resolve;
    });
  });
  function Links() {
    const [items, setItems] = useState(
      makePhotos(3).map((photo) => ({
        ...photo,
        urls_expires_at: "2000-01-01T00:00:00Z",
      })),
    );
    usePhotoLinks("event", items, setItems, [1], renew);
    return <output>{items[0].thumbnail_url}</output>;
  }
  root.render(<Links />);
  await vi.waitFor(() =>
    expect(container.querySelector("output")).not.toBeNull(),
  );
  expect(renew).not.toHaveBeenCalled();
  visibility = "visible";
  document.dispatchEvent(new Event("visibilitychange"));
  await vi.waitFor(() => expect(renew).toHaveBeenCalledTimes(1));
  expect(renew.mock.calls[0][0]).toEqual([1]);
  document.dispatchEvent(new Event("visibilitychange"));
  expect(renew).toHaveBeenCalledTimes(1);
  finish({
    thumbnail_urls: { "1": "renewed-thumbnail" },
    urls_expires_at: "2099-01-01T00:00:00Z",
  });
  await vi.waitFor(() =>
    expect(container.querySelector("output")?.textContent).toBe(
      "renewed-thumbnail",
    ),
  );
  document.dispatchEvent(new Event("visibilitychange"));
  expect(renew).toHaveBeenCalledTimes(1);
  vi.restoreAllMocks();
});
