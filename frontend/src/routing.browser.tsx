import { afterEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import { Admin, CustomRoutes, testDataProvider } from "react-admin";
import {
  createMemoryRouter,
  Link,
  Route,
  RouterProvider,
} from "react-router-dom";
import { createAppRoutes } from "./routing";
import { Translations } from "./components/Translations/Translations";
import { TranslationDraftProvider } from "./components/Translations/drafts";

vi.mock("./components/Translations/api", () => ({
  translationRequest: async () => ({
    resources: [],
    aiEnabled: false,
    dailyBudget: 1000000,
    usage: { day: "2026-11-02", used: 0, reserved: 0 },
  }),
}));

let root: Root;
let container: HTMLDivElement;
let router: ReturnType<typeof createMemoryRouter>;
afterEach(() => {
  root.unmount();
  router.dispose();
  container.remove();
});
function render(path: string) {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  const admin = (
    <Admin
      basename="/admin"
      dataProvider={testDataProvider()}
      disableTelemetry
      dashboard={() => <Link to="/admin/translations">Open translations</Link>}
      layout={({ children }) => (
        <TranslationDraftProvider>{children}</TranslationDraftProvider>
      )}
    >
      <CustomRoutes>
        <Route path="/translations" element={<Translations />} />
      </CustomRoutes>
    </Admin>
  );
  router = createMemoryRouter(createAppRoutes(admin), {
    initialEntries: [path],
  });
  root.render(<RouterProvider router={router} />);
}

test("opens the translations deep link through the real admin router", async () => {
  render("/admin/translations");
  await expect
    .element(page.getByRole("heading", { name: "Translations" }))
    .toBeVisible();
  await expect
    .element(page.getByText("AI generation is disabled.", { exact: false }))
    .toBeVisible();
  expect(router.state.location.pathname).toBe("/admin/translations");
});

test("redirects the root to the admin and preserves its prefix during navigation", async () => {
  render("/");
  await expect
    .element(page.getByRole("link", { name: "Open translations" }))
    .toBeVisible();
  expect(router.state.location.pathname).toBe("/admin");
  await page.getByRole("link", { name: "Open translations" }).click();
  await expect
    .element(page.getByRole("heading", { name: "Translations" }))
    .toBeVisible();
  expect(router.state.location.pathname).toBe("/admin/translations");
});
