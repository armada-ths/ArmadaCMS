import { afterEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import {
  AdminContext,
  Datagrid,
  type GetListParams,
  TextField,
  testDataProvider,
} from "react-admin";
import { ResourceList } from "./ResourceList";

let root: Root | undefined;
let container: HTMLDivElement | undefined;
afterEach(() => {
  root?.unmount();
  container?.remove();
});

test("search filters all records, resets pagination, preserves fixed filters, and clears", async () => {
  const getList = vi
    .fn()
    .mockImplementation(async (_resource: string, params: GetListParams) => ({
      data: params.filter.q === "missing" ? [] : [{ id: 1, name: "Armada" }],
      total: params.filter.q ? (params.filter.q === "missing" ? 0 : 1) : 50,
    }));
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  root.render(
    <AdminContext dataProvider={testDataProvider({ getList })}>
      <ResourceList
        resource="programs"
        storeKey={false}
        disableSyncWithLocation
        filter={{ enabled: true }}
      >
        <Datagrid bulkActionButtons={false}>
          <TextField source="name" />
        </Datagrid>
      </ResourceList>
    </AdminContext>,
  );
  await vi.waitFor(() => expect(getList).toHaveBeenCalled());
  const search = page.getByRole("textbox", { name: "ra.action.search" });
  await expect.element(search).toBeVisible();
  await page.getByRole("button", { name: "Go to next page" }).click();
  await vi.waitFor(() =>
    expect(getList.mock.lastCall?.[1].pagination.page).toBe(2),
  );
  await search.fill("Armada");
  await vi.waitFor(() => {
    const params = getList.mock.lastCall?.[1];
    expect(params.filter).toEqual({ enabled: true, q: "Armada" });
    expect(params.pagination.page).toBe(1);
  });
  await search.fill("missing");
  await vi.waitFor(() =>
    expect(getList.mock.lastCall?.[1].filter.q).toBe("missing"),
  );
  await expect.element(search).toBeVisible();
  await search.fill("");
  await vi.waitFor(() => {
    const filter = getList.mock.lastCall?.[1].filter;
    expect(filter.enabled).toBe(true);
    expect(filter.q || "").toBe("");
  });
});
