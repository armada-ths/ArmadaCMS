import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { EventroSyncReview } from "./EventroSyncReview";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  notify: vi.fn(),
}));
vi.mock("../../context/axiosInstance", () => ({
  default: { get: mocks.get, post: mocks.post },
}));
vi.mock("react-admin", () => ({ useNotify: () => mocks.notify }));

let container: HTMLDivElement, root: Root, client: QueryClient;
beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
  mocks.notify.mockReset();
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
});
afterEach(() => {
  root.unmount();
  container.remove();
  client.clear();
});

function render(perms = ["eventrosync.access", "profiles.edit"]) {
  root.render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <EventroSyncReview perms={perms} syncVersion={null} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

const row = {
  resource: "profiles",
  id: 7,
  name: "Member Example",
  missing: [],
  unmappedProducts: [],
  fields: [
    {
      field: "title",
      cms: "CMS title",
      eventro: "Eventro title",
      conflict: true,
    },
  ],
};

test("reviews and applies one field with both expected values", async () => {
  mocks.get
    .mockResolvedValueOnce({ data: [row] })
    .mockResolvedValue({ data: [] });
  mocks.post.mockResolvedValue({ data: { success: true } });
  render();
  await page
    .getByRole("button", { name: "Review sync and data quality" })
    .click();
  await expect
    .element(page.getByText("CMS title", { exact: true }))
    .toBeVisible();
  await expect
    .element(page.getByText("Eventro title", { exact: true }))
    .toBeVisible();
  await page.getByRole("button", { name: "Use Eventro" }).click();
  await vi.waitFor(() =>
    expect(mocks.post).toHaveBeenCalledWith("/eventrosyncreview", {
      resource: "profiles",
      recordId: 7,
      field: "title",
      choice: "eventro",
      cms: "CMS title",
      eventro: "Eventro title",
    }),
  );
  await expect.element(page.getByText("No items need review.")).toBeVisible();
});

test("keeping CMS removes the resolved item from review", async () => {
  mocks.get
    .mockResolvedValueOnce({ data: [row] })
    .mockResolvedValue({ data: [] });
  mocks.post.mockResolvedValue({ data: { success: true } });
  render();
  await page
    .getByRole("button", { name: "Review sync and data quality" })
    .click();
  await page.getByRole("button", { name: "Keep CMS" }).click();
  await vi.waitFor(() =>
    expect(mocks.post).toHaveBeenCalledWith(
      "/eventrosyncreview",
      expect.objectContaining({ choice: "cms", field: "title" }),
    ),
  );
  await expect.element(page.getByText("No items need review.")).toBeVisible();
  await expect
    .element(page.getByText("CMS title", { exact: true }))
    .not.toBeInTheDocument();
  await page.getByRole("button", { name: "Close", exact: true }).click();
  await page
    .getByRole("button", { name: "Review sync and data quality" })
    .click();
  await expect.element(page.getByText("No items need review.")).toBeVisible();
});

test("failed review leaves correction visible and re-enables actions", async () => {
  mocks.get.mockResolvedValue({ data: [row] });
  mocks.post.mockRejectedValue(new Error("409 values changed"));
  render();
  await page
    .getByRole("button", { name: "Review sync and data quality" })
    .click();
  await page.getByRole("button", { name: "Keep CMS" }).click();
  await vi.waitFor(() =>
    expect(mocks.notify).toHaveBeenCalledWith(
      expect.stringContaining("Refresh the review"),
      { type: "error" },
    ),
  );
  await expect
    .element(page.getByText("CMS title", { exact: true }))
    .toBeVisible();
  await expect
    .element(page.getByRole("button", { name: "Keep CMS" }))
    .toBeEnabled();
});

test("does not offer review when no resource can be edited", async () => {
  render(["eventrosync.access"]);
  await vi.waitFor(() => expect(container.textContent).toBe(""));
  expect(mocks.get).not.toHaveBeenCalled();
});
