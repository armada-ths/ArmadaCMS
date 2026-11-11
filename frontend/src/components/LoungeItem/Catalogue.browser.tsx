import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import {
  AdminContext,
  AdminUI,
  Notification,
  Resource,
  testDataProvider,
} from "react-admin";
import { QueryClient } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import type { AuthProvider } from "react-admin";
import { Catalogue } from "./catalogue";
import { LoungeItemList } from "./LoungeItemList";
import { LoungeItemCreate } from "./LoungeItemCreate";
import { LoungeItemEdit } from "./LoungeItemEdit";

const state = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("./catalogue", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./catalogue")>();
  return { ...actual, loungeRequest: state.request };
});
let container: HTMLDivElement, root: Root;
const canAccess = vi.fn().mockResolvedValue(true);
const auth: AuthProvider = {
  login: async () => {},
  logout: async () => {},
  checkAuth: async () => {},
  checkError: async () => {},
  getPermissions: async () => ["*"],
  canAccess,
};
const fixture: Catalogue = {
  revision: 0,
  categories: [
    { id: 1, name: "Drinks", sortOrder: 0 },
    { id: 2, name: "Snacks", sortOrder: 1 },
  ],
  items: [
    { id: 1, categoryId: 1, name: "Coffee", sortOrder: 0, soldOut: false },
    { id: 2, categoryId: 1, name: "Tea", sortOrder: 1, soldOut: false },
  ],
};
beforeEach(() => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  state.request.mockReset().mockResolvedValue(fixture);
  canAccess.mockReset().mockResolvedValue(true);
});
afterEach(() => {
  root.unmount();
  container.remove();
});
function render() {
  root.render(
    <AdminContext authProvider={auth} dataProvider={testDataProvider()}>
      <LoungeItemList />
      <Notification />
    </AdminContext>,
  );
}

test("requires confirmation with item count before cascading category deletion", async () => {
  render();
  await page
    .getByRole("button", { name: "Delete category", exact: true })
    .first()
    .click();
  await expect
    .element(
      page.getByText("Drinks: this will delete 2 items.", { exact: false }),
    )
    .toBeVisible();
  expect(state.request).not.toHaveBeenCalledWith(
    "lounge-categories/1",
    "DELETE",
    undefined,
  );
  await page.getByRole("button", { name: "Delete category and items" }).click();
  await vi.waitFor(() =>
    expect(state.request).toHaveBeenCalledWith(
      "lounge-categories/1",
      "DELETE",
      undefined,
    ),
  );
});

test("creates localized categories without numeric ordering or pictures", async () => {
  render();
  await page.getByRole("button", { name: "Add category" }).click();
  await page.getByRole("textbox", { name: "Name (English)" }).fill("Fruit");
  await page.getByRole("textbox", { name: "Name (Swedish)" }).fill("Frukt");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await vi.waitFor(() =>
    expect(state.request).toHaveBeenCalledWith("lounge-categories", "POST", {
      name: "Fruit",
      nameSv: "Frukt",
    }),
  );
  expect(container.textContent).not.toContain("Display order");
  expect(container.querySelector("img")).toBeNull();
});

test("keyboard drag saves a complete category order with its revision", async () => {
  render();
  const handle = page.getByRole("button", { name: "Move Snacks" });
  await expect.element(handle).toBeVisible();
  const node = handle.element();
  node.focus();
  node.dispatchEvent(
    new KeyboardEvent("keydown", { code: "Space", key: " ", bubbles: true }),
  );
  await new Promise((resolve) => setTimeout(resolve, 50));
  document.dispatchEvent(
    new KeyboardEvent("keydown", {
      code: "ArrowUp",
      key: "ArrowUp",
      bubbles: true,
    }),
  );
  await new Promise((resolve) => setTimeout(resolve, 50));
  document.dispatchEvent(
    new KeyboardEvent("keydown", { code: "Space", key: " ", bubbles: true }),
  );
  await vi.waitFor(() =>
    expect(state.request).toHaveBeenCalledWith("lounge-catalogue", "PUT", {
      revision: 0,
      categories: [
        { id: 2, itemIds: [] },
        { id: 1, itemIds: [1, 2] },
      ],
    }),
  );
});

test("hides management controls from view-only users", async () => {
  canAccess.mockImplementation(
    async ({ action }: { action: string }) => action === "list",
  );
  render();
  await expect.element(page.getByText("Drinks", { exact: true })).toBeVisible();
  expect(container.querySelector('[aria-label="Move Drinks"]')).toBeNull();
  expect(container.textContent).not.toContain("Add category");
});

test("keyboard drag reorders items inside a category", async () => {
  render();
  const handle = page.getByRole("button", { name: "Move Coffee" });
  await expect.element(handle).toBeVisible();
  handle.element().focus();
  handle
    .element()
    .dispatchEvent(
      new KeyboardEvent("keydown", { code: "Space", key: " ", bubbles: true }),
    );
  await new Promise((resolve) => setTimeout(resolve, 50));
  document.dispatchEvent(
    new KeyboardEvent("keydown", {
      code: "ArrowDown",
      key: "ArrowDown",
      bubbles: true,
    }),
  );
  await new Promise((resolve) => setTimeout(resolve, 50));
  document.dispatchEvent(
    new KeyboardEvent("keydown", { code: "Space", key: " ", bubbles: true }),
  );
  await vi.waitFor(() =>
    expect(state.request).toHaveBeenCalledWith("lounge-catalogue", "PUT", {
      revision: 0,
      categories: [
        { id: 1, itemIds: [2, 1] },
        { id: 2, itemIds: [] },
      ],
    }),
  );
});

test("keyboard drag can move an item into an empty category", async () => {
  render();
  const handle = page.getByRole("button", { name: "Move Tea" });
  await expect.element(handle).toBeVisible();
  handle.element().focus();
  handle
    .element()
    .dispatchEvent(
      new KeyboardEvent("keydown", { code: "Space", key: " ", bubbles: true }),
    );
  await new Promise((resolve) => setTimeout(resolve, 50));
  document.dispatchEvent(
    new KeyboardEvent("keydown", {
      code: "ArrowDown",
      key: "ArrowDown",
      bubbles: true,
    }),
  );
  await new Promise((resolve) => setTimeout(resolve, 50));
  document.dispatchEvent(
    new KeyboardEvent("keydown", { code: "Space", key: " ", bubbles: true }),
  );
  await vi.waitFor(() =>
    expect(state.request).toHaveBeenCalledWith("lounge-catalogue", "PUT", {
      revision: 0,
      categories: [
        { id: 1, itemIds: [1] },
        { id: 2, itemIds: [2] },
      ],
    }),
  );
});

test("pointer drag reorders items", async () => {
  render();
  const handle = page.getByRole("button", { name: "Move Coffee" });
  const target = page.getByRole("button", { name: "Move Tea" });
  await expect.element(handle).toBeVisible();
  const start = handle.element().getBoundingClientRect();
  const end = target.element().getBoundingClientRect();
  const pointer = (type: string, x: number, y: number) =>
    new PointerEvent(type, {
      bubbles: true,
      pointerId: 1,
      pointerType: "mouse",
      isPrimary: true,
      button: 0,
      buttons: type === "pointerup" ? 0 : 1,
      clientX: x,
      clientY: y,
    });
  const x = start.x + start.width / 2;
  const y = start.y + start.height / 2;
  handle.element().dispatchEvent(pointer("pointerdown", x, y));
  document.dispatchEvent(pointer("pointermove", x, y + 10));
  await new Promise((resolve) => setTimeout(resolve, 50));
  document.dispatchEvent(pointer("pointermove", x, end.y + end.height / 2));
  await new Promise((resolve) => setTimeout(resolve, 50));
  document.dispatchEvent(pointer("pointerup", x, end.y + end.height / 2));
  await vi.waitFor(() =>
    expect(state.request).toHaveBeenCalledWith("lounge-catalogue", "PUT", {
      revision: 0,
      categories: [
        { id: 1, itemIds: [2, 1] },
        { id: 2, itemIds: [] },
      ],
    }),
  );
});

test("failed reorder explicitly reports the conflict and reloads the canonical catalogue", async () => {
  state.request.mockImplementation(async (path: string, method?: string) => {
    if (method === "PUT") throw new Error("Catalogue revision conflict");
    if (path === "lounge-catalogue") return fixture;
    throw new Error("Unexpected request");
  });
  render();
  const handle = page.getByRole("button", { name: "Move Snacks" });
  await expect.element(handle).toBeVisible();
  handle.element().focus();
  handle
    .element()
    .dispatchEvent(
      new KeyboardEvent("keydown", { code: "Space", key: " ", bubbles: true }),
    );
  await new Promise((resolve) => setTimeout(resolve, 50));
  document.dispatchEvent(
    new KeyboardEvent("keydown", {
      code: "ArrowUp",
      key: "ArrowUp",
      bubbles: true,
    }),
  );
  await new Promise((resolve) => setTimeout(resolve, 50));
  document.dispatchEvent(
    new KeyboardEvent("keydown", { code: "Space", key: " ", bubbles: true }),
  );
  await expect
    .element(
      page.getByText("Could not save catalogue changes.", { exact: false }),
    )
    .toBeVisible();
  await vi.waitFor(() =>
    expect(
      state.request.mock.calls.filter(
        ([path, method]) => path === "lounge-catalogue" && method === undefined,
      ),
    ).toHaveLength(2),
  );
});

function renderItemFlow(
  path: string,
  save: () => Promise<{ data: Catalogue["items"][number] }>,
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  client.setQueryData(["lounge", "catalogue"], fixture);
  root.render(
    <MemoryRouter initialEntries={[path]}>
      <AdminContext
        authProvider={auth}
        queryClient={client}
        dataProvider={testDataProvider({
          create: vi.fn().mockImplementation(save),
          update: vi.fn().mockImplementation(save),
          getOne: vi.fn().mockResolvedValue({ data: fixture.items[0] }),
          getList: vi.fn().mockResolvedValue({
            data: fixture.categories,
            total: fixture.categories.length,
          }),
          getMany: vi.fn().mockResolvedValue({ data: fixture.categories }),
        })}
      >
        <AdminUI>
          <Resource
            name="lounge-items"
            list={LoungeItemList}
            create={LoungeItemCreate}
            edit={LoungeItemEdit}
          />
          <Resource name="lounge-categories" />
        </AdminUI>
      </AdminContext>
    </MemoryRouter>,
  );
}

test.each(["create", "edit"])(
  "successful item %s returns to the refreshed drag-and-drop overview",
  async (mode) => {
    const item = { ...fixture.items[0], name: "Espresso" };
    const save = vi.fn().mockResolvedValue({ data: item });
    state.request.mockResolvedValue({
      ...fixture,
      items: [item, fixture.items[1]],
    });
    renderItemFlow(
      mode === "create" ? "/lounge-items/create" : "/lounge-items/1",
      save,
    );
    const name = page.getByRole("textbox", { name: "Name (English)" });
    await expect.element(name).toBeVisible();
    await name.fill("Espresso");
    if (mode === "create") {
      await page.getByRole("combobox", { name: /Category/ }).click();
      await page.getByRole("option", { name: "Drinks" }).click();
    }
    await page.getByRole("button", { name: /save/i }).click();
    await expect
      .element(page.getByRole("button", { name: "Move Espresso" }))
      .toBeVisible();
    expect(save).toHaveBeenCalledTimes(1);
    expect(state.request).toHaveBeenCalledWith("lounge-catalogue");
  },
);

test("failed item creation stays in the form with entered values", async () => {
  const save = vi.fn().mockRejectedValue(new Error("Save failed"));
  renderItemFlow("/lounge-items/create", save);
  const name = page.getByRole("textbox", { name: "Name (English)" });
  await expect.element(name).toBeVisible();
  await name.fill("Espresso");
  await page.getByRole("combobox", { name: /Category/ }).click();
  await page.getByRole("option", { name: "Drinks" }).click();
  await page.getByRole("button", { name: /save/i }).click();
  await expect
    .element(page.getByText("Save failed", { exact: true }))
    .toBeVisible();
  await expect.element(name).toHaveValue("Espresso");
  expect(state.request).not.toHaveBeenCalled();
});

test.each(["create", "edit"])(
  "can leave the item %s screen without saving",
  async (mode) => {
    const save = vi.fn().mockResolvedValue({ data: fixture.items[0] });
    renderItemFlow(
      mode === "create" ? "/lounge-items/create" : "/lounge-items/1",
      save,
    );
    await expect
      .element(page.getByRole("textbox", { name: "Name (English)" }))
      .toBeVisible();
    await page.getByRole("link", { name: "Back to catalogue" }).click();
    await expect
      .element(page.getByRole("button", { name: "Move Coffee" }))
      .toBeVisible();
    await expect
      .element(page.getByText("Lounge catalogue", { exact: true }).last())
      .toBeVisible();
    await expect
      .element(
        page.getByText(
          "Arrange categories and items to set their order on the order page.",
          { exact: true },
        ),
      )
      .toBeVisible();
    expect(save).not.toHaveBeenCalled();
  },
);
