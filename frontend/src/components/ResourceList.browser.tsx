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
import { RESOURCE_FILTERS } from "./resourceFilters";

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
        resource="featureflags"
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

const mountList = (resource: string) => {
  const getList = vi
    .fn()
    .mockResolvedValue({ data: [{ id: 1, name: "Armada" }], total: 1 });
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  root.render(
    <AdminContext dataProvider={testDataProvider({ getList })}>
      <ResourceList
        resource={resource}
        storeKey={false}
        disableSyncWithLocation
      >
        <Datagrid bulkActionButtons={false}>
          <TextField source="name" />
        </Datagrid>
      </ResourceList>
    </AdminContext>,
  );
  return getList;
};

const addFilter = async (label: string) => {
  await page.getByRole("button", { name: "ra.action.add_filter" }).click();
  await page
    .getByRole("menuitemcheckbox", { name: label, exact: true })
    .click();
};

test.each(
  Object.entries(RESOURCE_FILTERS)
    .filter(([, filters]) => filters.length)
    .map(([resource]) => resource),
)(
  "%s exposes column filters and combines numeric ID with search",
  async (resource) => {
    const getList = mountList(resource);
    await vi.waitFor(() => expect(getList).toHaveBeenCalled());
    await page
      .getByRole("textbox", { name: "ra.action.search" })
      .fill("Armada");
    await vi.waitFor(() =>
      expect(getList.mock.lastCall?.[1].filter.q).toBe("Armada"),
    );
    await addFilter("ID");
    await page.getByRole("spinbutton", { name: "ID", exact: true }).fill("7");
    await vi.waitFor(() =>
      expect(getList.mock.lastCall?.[1].filter).toEqual({ q: "Armada", id: 7 }),
    );
  },
);

test("text column values are sent separately from free-text search and can be removed", async () => {
  const getList = mountList("programs");
  await vi.waitFor(() => expect(getList).toHaveBeenCalled());
  await addFilter("Name");
  await page
    .getByRole("textbox", { name: "Name", exact: true })
    .fill("Computer Science");
  await vi.waitFor(() =>
    expect(getList.mock.lastCall?.[1].filter).toEqual({
      name: "Computer Science",
    }),
  );
  await page.getByRole("button", { name: "ra.action.remove_filter" }).click();
  await vi.waitFor(() => expect(getList.mock.lastCall?.[1].filter).toEqual({}));
});

test("false remains an active boolean filter", async () => {
  const getList = mountList("featureflags");
  await vi.waitFor(() => expect(getList).toHaveBeenCalled());
  await addFilter("Enabled");
  await page.getByRole("combobox", { name: "Enabled", exact: true }).click();
  await page
    .getByRole("option", { name: "ra.boolean.false", exact: true })
    .click();
  await vi.waitFor(() =>
    expect(getList.mock.lastCall?.[1].filter).toEqual({ enabled: false }),
  );
});

test("timestamp filters use the CMS date-time picker and send UTC values", async () => {
  const getList = mountList("customusers");
  await vi.waitFor(() => expect(getList).toHaveBeenCalled());
  await addFilter("Created at");
  const input = page.getByLabelText("Created at", { exact: true });
  await expect.element(input).toHaveAttribute("type", "datetime-local");
  const localValue = "2026-11-01T13:30";
  await input.fill(localValue);
  await vi.waitFor(() =>
    expect(getList.mock.lastCall?.[1].filter).toEqual({
      created_at: new Date(localValue).toISOString(),
    }),
  );
  await input.fill("");
  // The standard CMS date picker applies clearing when focus leaves the field.
  await page.getByRole("textbox", { name: "ra.action.search" }).click();
  await vi.waitFor(() => expect(getList.mock.lastCall?.[1].filter).toEqual({}));
});

test("fair date filters use a date-only picker", async () => {
  const getList = mountList("fairdates");
  await vi.waitFor(() => expect(getList).toHaveBeenCalled());
  await addFilter("IR start");
  const input = page.getByLabelText("IR start", { exact: true });
  await expect.element(input).toHaveAttribute("type", "date");
  await input.fill("2026-11-01");
  await vi.waitFor(() =>
    expect(getList.mock.lastCall?.[1].filter).toEqual({
      ir_start: "2026-11-01",
    }),
  );
});

test("predefined choices send their stored column value", async () => {
  const getList = mountList("exhibitors");
  await vi.waitFor(() => expect(getList).toHaveBeenCalled());
  await addFilter("Tier");
  await page.getByRole("combobox", { name: "Tier", exact: true }).click();
  await page.getByRole("option", { name: "Gold", exact: true }).click();
  await vi.waitFor(() =>
    expect(getList.mock.lastCall?.[1].filter).toEqual({ tier: "Gold" }),
  );
});
