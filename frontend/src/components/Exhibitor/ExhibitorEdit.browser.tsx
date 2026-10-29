import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import { AdminContext, testDataProvider } from "react-admin";
import { ExhibitorEdit } from "./ExhibitorEdit";

const categories = {
  industries: [{ id: 11, name: "Engineering" }],
  programs: [{ id: 22, name: "Computer Science" }],
  employments: [{ id: 33, name: "Internship" }],
};
const record = {
  id: 1,
  name: "Example company",
  type: "company",
  fairLocation: "Nymble",
  ...categories,
};
let container: HTMLDivElement, root: Root;
beforeEach(() => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});
afterEach(() => {
  root.unmount();
  container.remove();
});

function renderRecord(data = record) {
  const getMany = vi.fn(
    async (resource: string, { ids }: { ids: (string | number)[] }) => {
      const entries = categories[resource as keyof typeof categories] ?? [];
      return {
        data: entries.filter((entry) =>
          ids.map(String).includes(String(entry.id)),
        ),
      };
    },
  );
  const update = vi.fn(async (_resource, { data }) => ({ data }));
  root.render(
    <AdminContext
      dataProvider={testDataProvider({
        getOne: vi.fn().mockResolvedValue({ data }),
        getList: vi.fn().mockImplementation(async (resource: string) => ({
          data: categories[resource as keyof typeof categories] ?? [],
          total: 1,
        })),
        getMany: vi.fn().mockImplementation(getMany),
        update,
      })}
    >
      <ExhibitorEdit
        id={1}
        resource="exhibitors"
        mutationMode="pessimistic"
        redirect={false}
      />
    </AdminContext>,
  );
  return { getMany, update };
}

test("opens embedded category records as selected choices and preserves them on save", async () => {
  const { getMany, update } = renderRecord();
  for (const resource of ["industries", "programs", "employments"] as const) {
    await expect
      .element(page.getByRole("combobox", { name: new RegExp(resource, "i") }))
      .toHaveTextContent(categories[resource][0].name);
  }
  await vi.waitFor(() => {
    expect(getMany).toHaveBeenCalled();
    for (const [, params] of getMany.mock.calls) {
      expect(
        params.ids.every(
          (id) => typeof id === "number" || typeof id === "string",
        ),
      ).toBe(true);
    }
  });
  await page
    .getByRole("textbox", { name: /fields\.name$/ })
    .fill("Renamed company");
  await page.getByRole("button", { name: /ra\.action\.save/i }).click();
  await vi.waitFor(() => expect(update).toHaveBeenCalled());
  const payload = update.mock.calls[0][1].data;
  expect(payload.industries).toEqual([{ id: 11 }]);
  expect(payload.programs).toEqual([{ id: 22 }]);
  expect(payload.employments).toEqual([{ id: 33 }]);
  // The original API/list record is untouched by form normalization.
  expect(record.industries).toEqual(categories.industries);
});
