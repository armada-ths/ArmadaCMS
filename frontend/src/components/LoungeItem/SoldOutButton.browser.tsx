import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import {
  AdminContext,
  Notification,
  RecordContextProvider,
  testDataProvider,
  useGetOne,
} from "react-admin";
import type { AuthProvider } from "react-admin";
import { SoldOutButton } from "./SoldOutButton";

let container: HTMLDivElement, root: Root;
let record = { id: 1, name: "Coffee", soldOut: false };
const update = vi.fn();
const canAccess = vi.fn().mockResolvedValue(true);
const authProvider: AuthProvider = {
  login: async () => {},
  logout: async () => {},
  checkAuth: async () => {},
  checkError: async () => {},
  getPermissions: async () => ["*"],
  canAccess,
};
function Record() {
  const { data } = useGetOne("lounge-items", { id: 1 });
  return (
    <RecordContextProvider value={data}>
      <SoldOutButton />
    </RecordContextProvider>
  );
}
function render() {
  root.render(
    <AdminContext
      authProvider={authProvider}
      dataProvider={testDataProvider({
        getOne: vi.fn().mockImplementation(async () => ({ data: record })),
        update,
      })}
    >
      <Record />
      <Notification />
    </AdminContext>,
  );
}
beforeEach(() => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  record = { id: 1, name: "Coffee", soldOut: false };
  update.mockReset();
  canAccess.mockReset().mockResolvedValue(true);
  update.mockImplementation(async (_resource, params) => {
    record = { ...record, ...params.data };
    return { data: record };
  });
});
afterEach(() => {
  root.unmount();
  container.remove();
});

test("persists sold-out and available status without resending other fields", async () => {
  render();
  await page.getByRole("button", { name: "Mark sold out" }).click();
  await vi.waitFor(() =>
    expect(update).toHaveBeenCalledWith(
      "lounge-items",
      expect.objectContaining({ id: 1, data: { soldOut: true } }),
    ),
  );
  await page.getByRole("button", { name: "Mark available" }).click();
  await vi.waitFor(() =>
    expect(update).toHaveBeenLastCalledWith(
      "lounge-items",
      expect.objectContaining({ id: 1, data: { soldOut: false } }),
    ),
  );
  expect(record.name).toBe("Coffee");
});

test("disables the button while persistence is pending and reports failure", async () => {
  let reject: (reason: Error) => void = () => {};
  update.mockImplementation(
    () =>
      new Promise((_resolve, rejectPromise) => {
        reject = rejectPromise;
      }),
  );
  render();
  const button = page.getByRole("button", { name: "Mark sold out" });
  await button.click();
  await expect.element(button).toBeDisabled();
  reject(new Error("Unavailable"));
  await expect
    .element(
      page.getByText("Could not change item availability. Please try again."),
    )
    .toBeVisible();
  await expect.element(button).toBeEnabled();
  expect(record.soldOut).toBe(false);
});

test("hides availability controls without edit permission", async () => {
  canAccess.mockResolvedValue(false);
  render();
  await vi.waitFor(() =>
    expect(canAccess).toHaveBeenCalledWith(
      expect.objectContaining({ action: "edit", resource: "lounge-items" }),
    ),
  );
  expect(container.querySelector("button")).toBeNull();
});
