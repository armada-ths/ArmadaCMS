import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import { AdminContext, testDataProvider } from "react-admin";
import { EventEdit } from "./EventEdit";
const imageUrl =
  "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7";
const event = {
  id: 1,
  name: "Lunch",
  location: "Nymble",
  imageUrl,
  eventStart: "2026-11-01T12:00:00Z",
  eventEnd: "2026-11-01T13:00:00Z",
  show: true,
  registrationRequired: false,
};
let container: HTMLDivElement;
let root: Root;
const update = vi.fn();
beforeEach(async () => {
  update
    .mockReset()
    .mockImplementation(async (_resource, params) => ({ data: params.data }));
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  root.render(
    <AdminContext
      dataProvider={testDataProvider({
        getOne: vi.fn().mockResolvedValue({ data: event }),
        update,
      })}
    >
      <EventEdit
        resource="events"
        id="1"
        mutationMode="pessimistic"
        redirect={false}
      />
    </AdminContext>,
  );
  await expect
    .element(page.getByRole("button", { name: "Remove photo" }))
    .toBeVisible();
});
afterEach(() => {
  root.unmount();
  container.remove();
});
const save = async () => {
  await page
    .getByRole("button", { name: "ra.action.save", exact: true })
    .click();
  await vi.waitFor(() => expect(update).toHaveBeenCalledOnce());
  return update.mock.calls[0][1].data;
};
const chooseFile = async () => {
  const input =
    container.querySelector<HTMLInputElement>('input[type="file"]')!;
  const transfer = new DataTransfer();
  transfer.items.add(
    new File(["photo"], "replacement.jpg", { type: "image/jpeg" }),
  );
  input.files = transfer.files;
  input.dispatchEvent(new Event("change", { bubbles: true }));
  await vi.waitFor(() =>
    expect(container.querySelector('img[src^="blob:"]')).not.toBeNull(),
  );
};
test("removes the saved cover and updates its preview before saving", async () => {
  await page.getByRole("button", { name: "Remove photo" }).click();
  await expect
    .element(page.getByText("Current image:"))
    .not.toBeInTheDocument();
  const data = await save();
  expect(data.imageUrl).toBeNull();
  expect(data.file).toBeNull();
});
test("submits a new file alongside the saved URL for backend precedence", async () => {
  await chooseFile();
  const data = await save();
  expect(data.file.rawFile.name).toBe("replacement.jpg");
  expect(data.imageUrl).toBe(imageUrl);
});
test("clears a pending upload when switching to a replacement URL", async () => {
  await chooseFile();
  await page.getByRole("radio", { name: "Enter link to photo" }).click();
  await page
    .getByRole("textbox", { name: "Photo URL" })
    .fill("https://example.com/new.jpg");
  const data = await save();
  expect(data.file).toBeNull();
  expect(data.imageUrl).toBe("https://example.com/new.jpg");
});
test("removal also clears a pending upload", async () => {
  await chooseFile();
  await page.getByRole("button", { name: "Remove photo" }).click();
  const data = await save();
  expect(data.imageUrl).toBeNull();
  expect(data.file).toBeNull();
});
