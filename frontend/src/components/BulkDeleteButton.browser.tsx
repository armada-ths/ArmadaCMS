import { afterEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import {
  AdminContext,
  Datagrid,
  List,
  Notification,
  TextField,
  testDataProvider,
} from "react-admin";
import { BulkDeleteButton } from "./BulkDeleteButton";
import { BulkDeleteError } from "../dataProvider";

let root: Root | undefined;
let container: HTMLDivElement | undefined;
afterEach(() => {
  root?.unmount();
  container?.remove();
});

test("partial deletion refreshes the list and retries only failed records", async () => {
  let records = [
    { id: 1, name: "First" },
    { id: 2, name: "Second" },
  ];
  const deleteMany = vi.fn().mockImplementation(async (_resource, { ids }) => {
    if (ids.includes(1)) {
      records = records.filter((record) => record.id !== 1);
      throw new BulkDeleteError([1], [2]);
    }
    records = [];
    return { data: ids };
  });
  const getList = vi.fn().mockImplementation(async () => ({
    data: records,
    total: records.length,
  }));
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  root.render(
    <AdminContext
      dataProvider={testDataProvider({
        getList,
        deleteMany,
      })}
    >
      <List resource="exhibitors" storeKey={false} disableSyncWithLocation>
        <Datagrid bulkActionButtons={<BulkDeleteButton />}>
          <TextField source="name" />
        </Datagrid>
      </List>
      <Notification />
    </AdminContext>,
  );
  await page.getByRole("checkbox", { name: "ra.action.select_all" }).click();
  await page
    .getByRole("button", { name: "ra.action.delete", exact: true })
    .click();
  // Opening the confirmation must not start the mutation.
  expect(deleteMany).not.toHaveBeenCalled();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "ra.action.confirm" })
    .click();
  await expect
    .element(
      page.getByText(
        "1 deleted. 1 could not be deleted. Retry the selected records.",
      ),
    )
    .toBeVisible();
  await expect
    .element(page.getByText("First", { exact: true }))
    .not.toBeInTheDocument();
  await expect.element(page.getByText("Second", { exact: true })).toBeVisible();
  await page
    .getByRole("button", { name: "ra.action.delete", exact: true })
    .click();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "ra.action.confirm" })
    .click();
  await vi.waitFor(() => expect(deleteMany).toHaveBeenCalledTimes(2));
  expect(deleteMany.mock.calls[1][1].ids).toEqual([2]);
  await expect
    .element(page.getByText("Second", { exact: true }))
    .not.toBeInTheDocument();
});
