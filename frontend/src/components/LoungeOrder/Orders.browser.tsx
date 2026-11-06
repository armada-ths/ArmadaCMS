import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import {
  AdminContext,
  Notification,
  RecordContextProvider,
  ResourceContextProvider,
  testDataProvider,
} from "react-admin";
import type { AuthProvider } from "react-admin";
import { LoungeOrderActions, LoungeOrderList } from "./LoungeOrders";

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
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  getList.mockClear();
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
    .element(page.getByText("Old category: 9007199254740992", { exact: true }))
    .toBeVisible();
  await vi.waitFor(() =>
    expect(getList).toHaveBeenCalledWith(
      "lounge-orders",
      expect.objectContaining({ filter: { isTest: false } }),
    ),
  );
  await page.getByRole("combobox", { name: "Company" }).click();
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
