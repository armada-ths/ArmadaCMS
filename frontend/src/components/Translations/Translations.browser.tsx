import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page } from "vitest/browser";
import { createRoot, type Root } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { Translations } from "./Translations";
import { TranslationDraftProvider } from "./drafts";
import type { TranslationRecord } from "./api";
const mocks = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock("./api", () => ({ translationRequest: mocks.request }));
vi.mock("react-admin", () => ({
  Title: () => null,
  useInput: vi.fn(),
  useBasename: () => "/admin",
  useBlocker: () => ({ state: "unblocked" }),
}));
let root: Root, container: HTMLDivElement, client: QueryClient;
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
  container.remove();
  client.clear();
  vi.restoreAllMocks();
});
function render() {
  root.render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <TranslationDraftProvider>
          <Translations />
        </TranslationDraftProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
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
