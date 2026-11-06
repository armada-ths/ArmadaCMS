import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import {
  AdminContext,
  Notification,
  RecordContextProvider,
  ResourceContextProvider,
  testDataProvider,
  memoryStore,
} from "react-admin";
import type { AuthProvider } from "react-admin";
import {
  LoungeOrderActions,
  LoungeOrderDetails,
  LoungeOrderList,
  LoungeOrderShowActions,
} from "./LoungeOrders";

const state = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("../LoungeItem/catalogue", async (importOriginal) => {
  const original =
    await importOriginal<typeof import("../LoungeItem/catalogue")>();
  return { ...original, loungeRequest: state.request };
});
const auth: AuthProvider = {
  login: async () => {},
  logout: async () => {},
  checkAuth: async () => {},
  checkError: async () => {},
  getPermissions: async () => ["*"],
  canAccess: async () => true,
};
let container: HTMLDivElement, root: Root;
const getList = vi.fn().mockResolvedValue({ data: [], total: 0 });
const update = vi
  .fn()
  .mockResolvedValue({ data: { id: 1, status: "completed" } });
beforeEach(() => {
  window.history.replaceState(
    null,
    "",
    window.location.pathname + window.location.search,
  );
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  getList.mockReset().mockResolvedValue({ data: [], total: 0 });
  update.mockClear();
  state.request.mockReset().mockImplementation(async (path: string) => {
    if (path === "lounge-orders/filter-options")
      return {
        companies: [{ id: 1, name: "Deleted company" }],
        categories: [{ id: 2, name: "Old category" }],
        items: [{ id: 3, name: "Deleted item" }],
      };
    if (path.startsWith("lounge-orders/totals"))
      return [
        {
          itemId: 3,
          categoryId: 2,
          itemName: "Deleted item",
          categoryName: "Old category",
          quantity: "9007199254740992",
        },
      ];
    return { success: true };
  });
});
afterEach(() => {
  root.unmount();
  container.remove();
});
function render(element: React.ReactNode) {
  root.render(
    <AdminContext
      store={memoryStore()}
      authProvider={auth}
      dataProvider={testDataProvider({ getList, update })}
    >
      <ResourceContextProvider value="lounge-orders">
        {element}
      </ResourceContextProvider>
      <Notification />
    </AdminContext>,
  );
}
test("totals cover the full dataset with exact large quantities and historical filter choices", async () => {
  render(<LoungeOrderList />);
  await expect
    .element(
      page.getByRole("heading", { name: "Old category 9007199254740992" }),
    )
    .toBeVisible();
  await vi.waitFor(() =>
    expect(getList).toHaveBeenCalledWith(
      "lounge-orders",
      expect.objectContaining({ filter: { isTest: false } }),
    ),
  );
  await page.getByRole("combobox", { name: /^Company/ }).click();
  await page.getByRole("option", { name: "Deleted company" }).click();
  await vi.waitFor(() =>
    expect(getList).toHaveBeenLastCalledWith(
      "lounge-orders",
      expect.objectContaining({ filter: { isTest: false, companyId: 1 } }),
    ),
  );
  await vi.waitFor(() =>
    expect(
      state.request.mock.calls.some(([path]) =>
        String(path).includes("companyId%22%3A1"),
      ),
    ).toBe(true),
  );
  await page.getByRole("combobox", { name: /Environment/ }).click();
  await page.getByRole("option", { name: "Test orders" }).click();
  await vi.waitFor(() =>
    expect(getList).toHaveBeenLastCalledWith(
      "lounge-orders",
      expect.objectContaining({ filter: { isTest: true, companyId: 1 } }),
    ),
  );
});
test("totals nest item quantities under their category total", async () => {
  state.request.mockImplementation(async (path: string) => {
    if (path === "lounge-orders/filter-options")
      return { companies: [], categories: [], items: [] };
    if (path.startsWith("lounge-orders/totals"))
      return [
        {
          itemId: 1,
          categoryId: 2,
          itemName: "Coffee",
          categoryName: "Beverages",
          quantity: "9007199254740992",
        },
        {
          itemId: 2,
          categoryId: 3,
          itemName: "Apple Pie",
          categoryName: "Sweets",
          quantity: "3",
        },
        {
          itemId: 3,
          categoryId: 2,
          itemName: "Lemonade",
          categoryName: "Beverages",
          quantity: "2",
        },
      ];
    throw new Error(`Unexpected request: ${path}`);
  });
  render(<LoungeOrderList />);
  const beverages = page.getByRole("region", {
    name: "Beverages 9007199254740994",
  });
  const sweets = page.getByRole("region", { name: "Sweets 3" });
  await expect.element(beverages).toBeVisible();
  await expect.element(sweets).toBeVisible();
  await expect
    .element(beverages.getByText("Coffee", { exact: true }))
    .toBeVisible();
  await expect
    .element(beverages.getByText("Lemonade", { exact: true }))
    .toBeVisible();
  await expect
    .element(sweets.getByText("Apple Pie", { exact: true }))
    .toBeVisible();
  const sections = container.querySelectorAll("section[aria-labelledby]");
  expect(sections).toHaveLength(2);
  expect(
    [...sections[0].querySelectorAll("dl > div")].map((row) => [
      row.querySelector("dt")?.textContent,
      row.querySelector("dd")?.textContent,
    ]),
  ).toEqual([
    ["Coffee", "9007199254740992"],
    ["Lemonade", "2"],
  ]);
  expect(sweets.element().querySelectorAll("dt")).toHaveLength(1);
});
test("empty filters show (none) and clearing a company removes its filter", async () => {
  render(<LoungeOrderList />);
  for (const name of ["Company", "Category", "Item", "Status"]) {
    const filter = page.getByRole("combobox", { name: new RegExp(`^${name}`) });
    await expect.element(filter).toHaveTextContent("(none)");
    await filter.click();
    await expect
      .element(
        page.getByRole("option", { name: "ra.action.clear_input_value" }),
      )
      .toBeVisible();
    await page
      .getByRole("option", { name: "ra.action.clear_input_value" })
      .click();
  }
  await page.getByRole("combobox", { name: /^Company/ }).click();
  await page.getByRole("option", { name: "Deleted company" }).click();
  await vi.waitFor(() =>
    expect(getList).toHaveBeenLastCalledWith(
      "lounge-orders",
      expect.objectContaining({ filter: { isTest: false, companyId: 1 } }),
    ),
  );
  await page.getByRole("combobox", { name: /^Company/ }).click();
  await page
    .getByRole("option", { name: "ra.action.clear_input_value" })
    .click();
  await vi.waitFor(() =>
    expect(getList).toHaveBeenLastCalledWith(
      "lounge-orders",
      expect.objectContaining({ filter: { isTest: false } }),
    ),
  );
  await page.getByRole("combobox", { name: /Environment/ }).click();
  await expect
    .element(page.getByRole("option", { name: "ra.action.clear_input_value" }))
    .toBeVisible();
});

test("statuses have consistent semantic colours in filters and order controls", async () => {
  render(<LoungeOrderList />);
  await page.getByRole("combobox", { name: /^Status/ }).click();
  for (const [status, color] of [
    ["completed", "Success"],
    ["pending", "Warning"],
    ["cancelled", "Error"],
  ]) {
    const option = page.getByRole("option", { name: status, exact: true });
    await expect.element(option).toBeVisible();
    expect(
      option.element().querySelector(`.MuiChip-color${color}`),
    ).not.toBeNull();
    if (status === "pending") {
      expect(
        getComputedStyle(option.element().querySelector(".MuiChip-root")!)
          .backgroundColor,
      ).toBe("rgb(251, 192, 45)");
    }
  }
  await page.getByRole("option", { name: "completed", exact: true }).click();
  expect(
    page
      .getByRole("combobox", { name: /^Status/ })
      .element()
      .querySelector(".MuiChip-colorSuccess"),
  ).not.toBeNull();
});

test("order details provide a back link to all orders", async () => {
  render(<LoungeOrderShowActions />);
  const back = page.getByRole("link", { name: "Back to all orders" });
  await expect.element(back).toBeVisible();
  expect(back.element().getAttribute("href")).toMatch(/\/lounge-orders$/);
});
test("order list uses coloured status badges for every order state", async () => {
  getList.mockResolvedValue({
    data: ["completed", "pending", "cancelled"].map((status, index) => ({
      id: index + 1,
      companyName: `Company ${index}`,
      status,
      deliveryStatus: "pending",
      isTest: true,
    })),
    total: 3,
  });
  render(<LoungeOrderList />);
  await expect
    .element(page.getByText("Company 2", { exact: true }))
    .toBeVisible();
  for (const [index, color] of ["Success", "Warning", "Error"].entries()) {
    const row = page
      .getByRole("row")
      .filter({ hasText: `Company ${index}` })
      .element();
    expect(row.querySelector(".column-status .MuiChip-root")).toHaveClass(
      `MuiChip-color${color}`,
    );
    expect(
      row.querySelector(".column-deliveryStatus .MuiChip-root"),
    ).toHaveClass("MuiChip-colorWarning");
  }
});
test.each([
  ["completed", "Success"],
  ["pending", "Warning"],
  ["cancelled", "Error"],
])(
  "order details and status editor colour %s consistently",
  async (status, color) => {
    render(
      <RecordContextProvider
        value={{
          id: 1,
          status,
          deliveryStatus: "sent",
          isTest: true,
          lines: [],
        }}
      >
        <LoungeOrderDetails />
      </RecordContextProvider>,
    );
    const editor = page.getByRole("combobox", { name: "Order status" });
    await expect.element(editor).toBeVisible();
    expect(container.querySelectorAll(`.MuiChip-color${color}`)).toHaveLength(
      2,
    );
    await editor.click();
    const option = page.getByRole("option", { name: status, exact: true });
    expect(option.element().querySelector(".MuiChip-root")).toHaveClass(
      `MuiChip-color${color}`,
    );
  },
);
test("order details place the full breakdown in a highlighted responsive panel", async () => {
  render(
    <RecordContextProvider
      value={{
        id: 6,
        companyName: "Sample Software AB",
        createdAt: "2026-11-05T13:13:00Z",
        status: "completed",
        deliveryStatus: "sent",
        isTest: true,
        lines: Array.from({ length: 12 }, (_, index) => ({
          id: index + 1,
          categoryName: "Beverages",
          itemName: `Sample drink ${index + 1}`,
          quantity: index + 1,
        })),
      }}
    >
      <LoungeOrderDetails />
    </RecordContextProvider>,
  );
  const panel = page.getByRole("region", { name: "Order breakdown" });
  await expect.element(panel).toBeVisible();
  await expect
    .element(panel.getByText("Sample drink 12", { exact: true }))
    .toBeVisible();
  expect(panel.element().querySelectorAll("tbody tr")).toHaveLength(12);
  const company = page
    .getByText("Sample Software AB", { exact: true })
    .element();
  const grid = panel.element().parentElement!;
  const wide = window.matchMedia("(min-width: 900px)").matches;
  const panelBounds = panel.element().getBoundingClientRect();
  const companyBounds = company.getBoundingClientRect();
  if (wide) {
    expect(panelBounds.left).toBeGreaterThan(companyBounds.right);
    expect(panelBounds.top).toBeLessThan(companyBounds.bottom);
  } else {
    expect(panelBounds.top).toBeGreaterThan(companyBounds.bottom);
  }
  grid.style.width = "350px";
  grid.style.gridTemplateColumns = "minmax(0, 1fr)";
  const narrowPanel = panel.element().getBoundingClientRect();
  expect(narrowPanel.top).toBeGreaterThan(
    company.getBoundingClientRect().bottom,
  );
  expect(narrowPanel.right).toBeLessThanOrEqual(
    grid.getBoundingClientRect().right,
  );
});
test("notification retry warns about duplicate messages and does not recreate the order", async () => {
  render(
    <RecordContextProvider
      value={{
        id: 1,
        status: "pending",
        deliveryStatus: "unknown",
        isTest: false,
      }}
    >
      <LoungeOrderActions />
    </RecordContextProvider>,
  );
  await page.getByRole("button", { name: "Retry Slack notification" }).click();
  await expect
    .element(page.getByText("The order is already saved.", { exact: false }))
    .toBeVisible();
  expect(state.request).not.toHaveBeenCalled();
  await page
    .getByRole("button", { name: "Retry notification", exact: true })
    .click();
  await vi.waitFor(() =>
    expect(state.request).toHaveBeenCalledWith("lounge-orders/1/retry", "POST"),
  );
  expect(update).not.toHaveBeenCalled();
});
