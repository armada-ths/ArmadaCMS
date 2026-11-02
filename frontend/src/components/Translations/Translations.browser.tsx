import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  createMemoryRouter,
  Link,
  Outlet,
  RouterProvider,
} from "react-router-dom";
import { Translations } from "./Translations";
import { TranslationDraftProvider } from "./drafts";
import type { TranslationRecord } from "./api";
const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("./api", () => ({ translationRequest: mocks.request }));
vi.mock("react-admin", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-admin")>()),
  Title: () => null,
  useInput: vi.fn(),
  useBasename: () => "/admin",
}));
let root: Root, container: HTMLDivElement, client: QueryClient;
let router: ReturnType<typeof createMemoryRouter> | undefined;
let current: TranslationRecord;
let enabled: boolean;
let conflict: boolean;
const resources = [
  {
    id: "highlightcards",
    label: "Highlight cards",
    aiAllowed: true,
    canEdit: true,
    missingRecords: 1,
  },
  {
    id: "exhibitors",
    label: "Exhibitors",
    aiAllowed: false,
    canEdit: true,
    missingRecords: 1,
  },
];
beforeEach(() => {
  enabled = true;
  conflict = false;
  current = {
    id: 7,
    label: "Welcome",
    fingerprint: "original-hash",
    fields: ["Title", "Subtitle"].map((id, i) => ({
      id,
      format: "text",
      source: i ? "Meet the Hosts" : "Welcome to Armada",
      published: null,
      status: "review",
      warnings: [],
      olderPolicy: i === 0,
      suggestion: {
        id: i + 10,
        version: 2,
        draft: i ? "Moot vardarna" : "Valkommen till Armada",
        generated: "AI draft",
        status: "draft",
        targetBefore: null,
        model: "test-model",
        promptVersion: "older-prompt",
        glossaryVersion: "glossary",
        createdAt: "2026-11-02T10:00:00Z",
      },
    })),
  };
  mocks.request.mockReset();
  mocks.request.mockImplementation(
    async (
      path: string,
      body?: { action?: string; fields?: { fieldId: string; text: string }[] },
    ) => {
      if (path === "")
        return {
          resources,
          aiEnabled: enabled,
          dailyBudget: 1000000,
          usage: { day: "2026-11-02", used: 123, reserved: 0 },
        };
      if (path.includes("/review")) {
        if (conflict)
          throw new Error(
            "Translation changed; reload the record before reviewing",
          );
        current = {
          ...current,
          fields: current.fields.map((field) => {
            const selected = body?.fields?.find((f) => f.fieldId === field.id);
            return selected
              ? {
                  ...field,
                  published: body?.action === "approve" ? selected.text : null,
                  suggestion: {
                    ...field.suggestion!,
                    status:
                      body?.action === "approve"
                        ? "approved"
                        : body?.action === "reject"
                          ? "rejected"
                          : "draft",
                    draft: selected.text,
                    version: 3,
                  },
                }
              : field;
          }),
        };
        return current;
      }
      if (path.includes("/selection")) return { ids: [7], batchSize: 50 };
      if (path.includes("/generate")) return [{ id: 1 }];
      return { records: [current], total: 1 };
    },
  );
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  vi.spyOn(window, "confirm").mockReturnValue(true);
});
afterEach(() => {
  root.unmount();
  router?.dispose();
  router = undefined;
  container.remove();
  client.clear();
  vi.restoreAllMocks();
});
function render() {
  router = createMemoryRouter(
    [
      {
        element: (
          <QueryClientProvider client={client}>
            <TranslationDraftProvider>
              <Link to="/admin/other">Leave translations</Link>
              <Link to="/admin/translations">Return to translations</Link>
              <Outlet />
            </TranslationDraftProvider>
          </QueryClientProvider>
        ),
        children: [
          { path: "/admin/translations", element: <Translations /> },
          { path: "/admin/other", element: <p>Other CMS page</p> },
        ],
      },
    ],
    { initialEntries: ["/admin/translations"] },
  );
  root.render(<RouterProvider router={router} />);
}
async function selectFields() {
  await page
    .getByRole("checkbox", { name: "Title (review)", exact: true })
    .click();
  await page
    .getByRole("checkbox", { name: "Subtitle (review)", exact: true })
    .click();
}
test("edits and approves selected fields with one versioned request per record", async () => {
  render();
  await expect
    .element(page.getByText("Welcome to Armada", { exact: true }))
    .toBeVisible();
  await expect
    .element(
      page.getByText(
        "This suggestion uses older translation instructions or glossary.",
        { exact: false },
      ),
    )
    .toBeVisible();
  await page
    .getByRole("textbox", { name: "Title (Swedish)" })
    .fill("Valkommen Armada");
  await selectFields();
  await page
    .getByRole("button", { name: "Approve selected", exact: true })
    .click();
  await vi.waitFor(() => {
    const calls = mocks.request.mock.calls.filter((call) =>
      String(call[0]).includes("/review"),
    );
    expect(calls).toHaveLength(1);
    expect(calls[0][1]).toEqual({
      action: "approve",
      fingerprint: "original-hash",
      fields: [
        {
          fieldId: "Title",
          suggestionId: 10,
          version: 2,
          text: "Valkommen Armada",
          targetBefore: null,
        },
        {
          fieldId: "Subtitle",
          suggestionId: 11,
          version: 2,
          text: "Moot vardarna",
          targetBefore: null,
        },
      ],
    });
  });
  await expect
    .element(
      page.getByText("Published Swedish: Valkommen Armada", { exact: true }),
    )
    .toBeVisible();
});
test("keeps unsaved edits across resource changes and after an approval conflict", async () => {
  conflict = true;
  render();
  await page
    .getByRole("textbox", { name: "Title (Swedish)" })
    .fill("Local edited draft");
  await page.getByRole("combobox", { name: /Resource/ }).click();
  await page
    .getByRole("option", { name: "Exhibitors (1)", exact: true })
    .click();
  await expect
    .element(
      page.getByText("Exhibitor descriptions belong to exhibitors.", {
        exact: false,
      }),
    )
    .toBeVisible();
  await expect
    .element(
      page.getByRole("button", { name: "Generate suggestions", exact: true }),
    )
    .toBeDisabled();
  await page.getByRole("combobox", { name: /Resource/ }).click();
  await page
    .getByRole("option", { name: "Highlight cards (1)", exact: true })
    .click();
  await expect
    .element(page.getByRole("textbox", { name: "Title (Swedish)" }))
    .toHaveValue("Local edited draft");
  await selectFields();
  await page
    .getByRole("button", { name: "Approve selected", exact: true })
    .click();
  await expect
    .element(
      page.getByText("Translation changed; reload the record before reviewing"),
    )
    .toBeVisible();
  await expect
    .element(page.getByRole("textbox", { name: "Title (Swedish)" }))
    .toHaveValue("Local edited draft");
  await page.getByRole("button", { name: "Discard local edits" }).click();
  await expect
    .element(page.getByRole("textbox", { name: "Title (Swedish)" }))
    .toHaveValue("Valkommen till Armada");
});
test("manual review works with AI disabled and saving a draft does not publish", async () => {
  enabled = false;
  render();
  await expect
    .element(
      page.getByRole("button", { name: "Generate suggestions", exact: true }),
    )
    .toBeDisabled();
  await selectFields();
  await page.getByRole("button", { name: "Save draft", exact: true }).click();
  await vi.waitFor(() =>
    expect(
      mocks.request.mock.calls.find((call) =>
        String(call[0]).includes("/review"),
      )?.[1].action,
    ).toBe("draft"),
  );
  expect(current.fields.every((f) => f.published === null)).toBe(true);
});
test("generation and regeneration are explicit requests", async () => {
  render();
  await expect
    .element(
      page.getByRole("button", { name: "Generate suggestions", exact: true }),
    )
    .toBeEnabled();
  expect(
    mocks.request.mock.calls.some((call) =>
      String(call[0]).includes("/generate"),
    ),
  ).toBe(false);
  await page
    .getByRole("button", { name: "Generate suggestions", exact: true })
    .click();
  await vi.waitFor(() =>
    expect(mocks.request).toHaveBeenCalledWith("/highlightcards/generate", {
      ids: [7],
      regenerate: false,
    }),
  );
  await page
    .getByRole("button", { name: "Regenerate suggestions", exact: true })
    .click();
  await vi.waitFor(() =>
    expect(mocks.request).toHaveBeenCalledWith("/highlightcards/generate", {
      ids: [7],
      regenerate: true,
    }),
  );
});

test("reuses the Markdown editor without publishing the saved draft", async () => {
  current.fields[0].format = "markdown";
  current.fields[0].source = "## Welcome";
  current.fields[0].suggestion!.draft = "## Valkommen";
  render();
  await page
    .getByRole("textbox", { name: "Title (Swedish)" })
    .fill("## Redigerat utkast");
  await page
    .getByRole("checkbox", { name: "Title (review)", exact: true })
    .click();
  await page.getByRole("button", { name: "Save draft", exact: true }).click();
  await vi.waitFor(() =>
    expect(current.fields[0].suggestion?.draft).toBe("## Redigerat utkast"),
  );
  expect(current.fields[0].published).toBe(null);
});

test("preserves edits across pages and warns before unloading", async () => {
  const original = mocks.request.getMockImplementation()!;
  mocks.request.mockImplementation(async (path: string, ...args: unknown[]) => {
    const result = await original(path, ...args);
    if (path.includes("?"))
      return {
        records: [
          path.includes("page=1")
            ? { ...current, id: 8, label: "Second page" }
            : current,
        ],
        total: 20,
      };
    return result;
  });
  render();
  await page
    .getByRole("textbox", { name: "Title (Swedish)" })
    .fill("Unsaved page edit");
  const event = new Event("beforeunload", { cancelable: true });
  window.dispatchEvent(event);
  expect(event.defaultPrevented).toBe(true);
  await page.getByRole("button", { name: "Go to next page" }).click();
  await expect
    .element(page.getByText("Second page", { exact: true }))
    .toBeVisible();
  await expect
    .element(page.getByRole("textbox", { name: "Title (Swedish)" }))
    .toHaveValue("Valkommen till Armada");
  await page.getByRole("button", { name: "Go to previous page" }).click();
  await expect
    .element(page.getByRole("textbox", { name: "Title (Swedish)" }))
    .toHaveValue("Unsaved page edit");
});
test("shows budget waiting and cancels pending generation explicitly", async () => {
  current.job = {
    id: 1,
    status: "budget_wait",
    outdated: true,
    notBefore: "2026-11-03T00:00:00Z",
    model: "test-model",
    promptVersion: "prompt",
    glossaryVersion: "glossary",
    createdAt: "2026-11-02T10:00:00Z",
  };
  render();
  await expect
    .element(page.getByText("budget wait", { exact: true }))
    .toBeVisible();
  await expect
    .element(page.getByText("The queued source has changed.", { exact: false }))
    .toBeVisible();
  await page.getByRole("button", { name: "Cancel generation" }).click();
  await vi.waitFor(() =>
    expect(mocks.request).toHaveBeenCalledWith(
      "/highlightcards/7/jobs",
      undefined,
      "DELETE",
    ),
  );
});

test("prevents edits from being lost while the selected fields are saved", async () => {
  const original = mocks.request.getMockImplementation()!;
  let finish!: () => void;
  const pending = new Promise<void>((resolve) => {
    finish = resolve;
  });
  mocks.request.mockImplementation(async (path: string, ...args: unknown[]) => {
    if (path.includes("/review")) await pending;
    return original(path, ...args);
  });
  render();
  await selectFields();
  await page.getByRole("button", { name: "Save draft", exact: true }).click();
  await expect
    .element(page.getByRole("textbox", { name: "Title (Swedish)" }))
    .toBeDisabled();
  finish();
  await expect
    .element(page.getByRole("textbox", { name: "Title (Swedish)" }))
    .toBeEnabled();
});

test("allows rejecting an outdated proposal by its original suggestion version", async () => {
  current.fields[0].status = "stale";
  current.fields[0].publishedOutdated = true;
  render();
  await expect
    .element(
      page.getByText(
        "Published Swedish is based on an older English original.",
        { exact: false },
      ),
    )
    .toBeVisible();
  await page
    .getByRole("checkbox", { name: "Title (stale)", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Reject selected", exact: true })
    .click();
  await vi.waitFor(() =>
    expect(mocks.request).toHaveBeenCalledWith(
      "/highlightcards/7/review",
      expect.objectContaining({
        action: "reject",
        fields: [
          expect.objectContaining({
            fieldId: "Title",
            suggestionId: 10,
            version: 2,
          }),
        ],
      }),
    ),
  );
});

test("blocks leaving unsaved translations and preserves drafts after confirmed navigation", async () => {
  render();
  await page
    .getByRole("textbox", { name: "Title (Swedish)" })
    .fill("Unsaved navigation edit");
  const confirm = vi.mocked(window.confirm);
  confirm.mockReturnValue(false);
  await page.getByRole("link", { name: "Leave translations" }).click();
  await vi.waitFor(() => expect(confirm).toHaveBeenCalledTimes(1));
  await expect
    .element(page.getByRole("textbox", { name: "Title (Swedish)" }))
    .toHaveValue("Unsaved navigation edit");
  expect(router?.state.location.pathname).toBe("/admin/translations");
  confirm.mockReturnValue(true);
  await page.getByRole("link", { name: "Leave translations" }).click();
  await expect.element(page.getByText("Other CMS page")).toBeVisible();
  expect(confirm).toHaveBeenCalledTimes(2);
  await page.getByRole("link", { name: "Return to translations" }).click();
  await expect
    .element(page.getByRole("textbox", { name: "Title (Swedish)" }))
    .toHaveValue("Unsaved navigation edit");
});

test("leaves translations without prompting when no edits are pending", async () => {
  render();
  await expect
    .element(page.getByText("Welcome to Armada", { exact: true }))
    .toBeVisible();
  await page.getByRole("link", { name: "Leave translations" }).click();
  await expect.element(page.getByText("Other CMS page")).toBeVisible();
  expect(window.confirm).not.toHaveBeenCalled();
});

test("selects an entire resource across pages and keeps local edits during bulk generation", async () => {
  const original = mocks.request.getMockImplementation()!;
  mocks.request.mockImplementation(async (path: string, ...args: unknown[]) => {
    if (path === "/highlightcards/selection")
      return {
        ids: Array.from({ length: 53 }, (_, i) => i + 1),
        batchSize: 50,
      };
    return original(path, ...args);
  });
  render();
  await page
    .getByRole("textbox", { name: "Title (Swedish)" })
    .fill("Local draft stays");
  await page
    .getByRole("textbox", { name: "Search translations" })
    .fill("Narrow filter");
  await page
    .getByRole("checkbox", {
      name: "Entire resource: Highlight cards",
      exact: true,
    })
    .click();
  await page
    .getByRole("button", {
      name: "Generate for selected resources (1)",
      exact: true,
    })
    .click();
  await expect
    .element(page.getByText("Checked 53 records;", { exact: false }))
    .toBeVisible();
  const calls = mocks.request.mock.calls.filter(([path]) =>
    String(path).includes("/generate"),
  );
  expect(calls).toHaveLength(2);
  expect(calls[0][1]).toEqual({
    ids: Array.from({ length: 50 }, (_, i) => i + 1),
    regenerate: false,
  });
  expect(calls[1][1]).toEqual({ ids: [51, 52, 53], regenerate: false });
  await expect
    .element(page.getByRole("textbox", { name: "Title (Swedish)" }))
    .toHaveValue("Local draft stays");
  expect(
    mocks.request.mock.calls.some(([path]) => String(path).includes("/review")),
  ).toBe(false);
});

test("selects all eligible resources while excluding exhibitor-owned descriptions", async () => {
  const original = mocks.request.getMockImplementation()!;
  mocks.request.mockImplementation(async (path: string, ...args: unknown[]) => {
    const result = await original(path, ...args);
    if (path === "")
      return {
        ...result,
        resources: [
          ...resources,
          {
            id: "teams",
            label: "Teams",
            aiAllowed: true,
            canEdit: true,
            missingRecords: 1,
          },
          {
            id: "profiles",
            label: "Profiles",
            aiAllowed: true,
            canEdit: false,
            missingRecords: 1,
          },
        ],
      };
    if (path === "/teams/selection") return { ids: [99], batchSize: 50 };
    return result;
  });
  render();
  await expect
    .element(
      page.getByRole("checkbox", {
        name: "Entire resource: Teams",
        exact: true,
      }),
    )
    .toBeVisible();
  await page
    .getByRole("checkbox", {
      name: "Select all eligible resources",
      exact: true,
    })
    .click();
  await expect
    .element(
      page.getByRole("checkbox", {
        name: "Entire resource: Highlight cards",
        exact: true,
      }),
    )
    .toBeChecked();
  await expect
    .element(
      page.getByRole("checkbox", {
        name: "Entire resource: Teams",
        exact: true,
      }),
    )
    .toBeChecked();
  await expect
    .element(
      page.getByRole("checkbox", {
        name: "Entire resource: Exhibitors",
        exact: true,
      }),
    )
    .toBeDisabled();
  await expect
    .element(
      page.getByRole("checkbox", {
        name: "Entire resource: Profiles",
        exact: true,
      }),
    )
    .toBeDisabled();
  await page
    .getByRole("button", {
      name: "Generate for selected resources (2)",
      exact: true,
    })
    .click();
  await expect
    .element(page.getByText("Checked 2 records;", { exact: false }))
    .toBeVisible();
  expect(
    mocks.request.mock.calls
      .filter(([path]) => String(path).includes("/generate"))
      .map(([path]) => path),
  ).toEqual(["/highlightcards/generate", "/teams/generate"]);
});

test("reports partial bulk failure and retains the selection for retry", async () => {
  const original = mocks.request.getMockImplementation()!;
  let writes = 0;
  mocks.request.mockImplementation(async (path: string, ...args: unknown[]) => {
    if (path === "/highlightcards/selection")
      return { ids: Array.from({ length: 3 }, (_, i) => i + 1), batchSize: 2 };
    if (path === "/highlightcards/generate" && ++writes === 2)
      throw new Error("Temporary network failure");
    return original(path, ...args);
  });
  render();
  await page
    .getByRole("checkbox", {
      name: "Entire resource: Highlight cards",
      exact: true,
    })
    .click();
  await page
    .getByRole("button", {
      name: "Generate for selected resources (1)",
      exact: true,
    })
    .click();
  await expect
    .element(
      page.getByText("Temporary network failure Queueing stopped;", {
        exact: false,
      }),
    )
    .toBeVisible();
  await expect
    .element(
      page.getByRole("checkbox", {
        name: "Entire resource: Highlight cards",
        exact: true,
      }),
    )
    .toBeChecked();
  await expect
    .element(
      page.getByRole("button", {
        name: "Generate for selected resources (1)",
        exact: true,
      }),
    )
    .toBeEnabled();
});

test("stops further resource queueing on request without cancelling accepted jobs", async () => {
  const original = mocks.request.getMockImplementation()!;
  mocks.request.mockImplementation(async (path: string, ...args: unknown[]) => {
    if (path === "/highlightcards/selection")
      return { ids: [1, 2, 3], batchSize: 1 };
    if (path === "/highlightcards/generate") {
      const signal = args[2] as AbortSignal;
      return new Promise((_, reject) =>
        signal.addEventListener(
          "abort",
          () => reject(new DOMException("Aborted", "AbortError")),
          { once: true },
        ),
      );
    }
    return original(path, ...args);
  });
  render();
  await page
    .getByRole("checkbox", {
      name: "Entire resource: Highlight cards",
      exact: true,
    })
    .click();
  await page
    .getByRole("button", {
      name: "Generate for selected resources (1)",
      exact: true,
    })
    .click();
  await page
    .getByRole("button", { name: "Stop queueing", exact: true })
    .click();
  await expect
    .element(
      page.getByText(
        "Queueing stopped. Requests already accepted remain queued.",
      ),
    )
    .toBeVisible();
  expect(
    mocks.request.mock.calls.filter(([path]) =>
      String(path).includes("/generate"),
    ),
  ).toHaveLength(1);
  expect(
    mocks.request.mock.calls.some(([path]) => String(path).includes("/jobs")),
  ).toBe(false);
});
