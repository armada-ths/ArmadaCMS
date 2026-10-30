import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { fetchJson, baseUpdate } = vi.hoisted(() => ({
  fetchJson: vi.fn(),
  baseUpdate: vi.fn(),
}));

vi.mock("react-admin", () => ({
  fetchUtils: { fetchJson },
  HttpError: class extends Error {},
}));
vi.mock("ra-data-simple-rest", () => ({
  default: () => ({ update: baseUpdate }),
}));
vi.mock("./context/globalApi", () => ({
  default: () => "https://cms.example.com/api/v1",
}));

import { BulkDeleteError, dataProvider } from "./dataProvider";

describe("bulk deletion", () => {
  it("limits 140 deletions to five active requests and accepts empty 204 responses", async () => {
    let active = 0;
    let peak = 0;
    fetchJson.mockImplementation(async () => {
      active++;
      peak = Math.max(peak, active);
      await new Promise((resolve) => setTimeout(resolve, 0));
      active--;
      return { status: 204, json: undefined };
    });
    const ids = Array.from({ length: 140 }, (_, i) => i + 1);
    await expect(
      dataProvider.deleteMany("exhibitors", { ids }),
    ).resolves.toEqual({ data: ids });
    expect(peak).toBe(5);
    expect(active).toBe(0);
    expect(fetchJson).toHaveBeenCalledTimes(140);
    expect(fetchJson).toHaveBeenCalledWith(
      "https://cms.example.com/api/v1/exhibitors/1",
      expect.objectContaining({
        method: "DELETE",
        headers: expect.any(Headers),
      }),
    );
    expect(fetchJson.mock.calls[0][1].headers.get("Authorization")).toBe(
      "Bearer test-token",
    );
  });

  it("waits for the queue and reports exactly which IDs succeeded or failed", async () => {
    const completed: string[] = [];
    fetchJson.mockImplementation(async (url: string) => {
      await new Promise((resolve) =>
        setTimeout(resolve, url.endsWith("/2") ? 5 : 0),
      );
      completed.push(url);
      if (url.endsWith("/2") || url.endsWith("/7")) throw new Error("500");
      return { status: 204 };
    });
    const ids = Array.from({ length: 12 }, (_, i) => i + 1);
    const error = await dataProvider
      .deleteMany("exhibitors", { ids })
      .catch((error: unknown) => error);
    expect(error).toBeInstanceOf(BulkDeleteError);
    expect(error).toMatchObject({
      deletedIds: ids.filter((id) => id !== 2 && id !== 7),
      failedIds: [2, 7],
    });
    expect(completed).toHaveLength(12);
  });

  it("reports total failure without returning a successful result", async () => {
    fetchJson.mockRejectedValue(new Error("500"));
    await expect(
      dataProvider.deleteMany("exhibitors", { ids: [1, 2] }),
    ).rejects.toMatchObject({ deletedIds: [], failedIds: [1, 2] });
  });

  it("handles empty selections and encodes string IDs", async () => {
    await expect(
      dataProvider.deleteMany("exhibitors", { ids: [] }),
    ).resolves.toEqual({ data: [] });
    expect(fetchJson).not.toHaveBeenCalled();
    await expect(
      dataProvider.deleteMany("exhibitors", { ids: ["a/b", "a/b"] }),
    ).resolves.toEqual({ data: ["a/b"] });
    expect(fetchJson).toHaveBeenCalledTimes(1);
    expect(fetchJson).toHaveBeenCalledWith(
      "https://cms.example.com/api/v1/exhibitors/a%2Fb",
      expect.anything(),
    );
  });
});

beforeEach(() => {
  vi.clearAllMocks();
  vi.stubGlobal("localStorage", { getItem: () => "test-token" });
  fetchJson.mockResolvedValue({ json: { id: 1 } });
});

afterEach(() => vi.unstubAllGlobals());

describe("timeline image updates", () => {
  const update = async (data: Record<string, unknown>) => {
    await dataProvider.update("timeline-entries", {
      id: 1,
      data,
      previousData: { id: 1, imageUrl: { src: "https://example.com/old.jpg" } },
    });
    expect(fetchJson).toHaveBeenCalledWith(
      "https://cms.example.com/api/v1/timeline-entries/1",
      expect.objectContaining({ method: "PUT" }),
    );
    return fetchJson.mock.calls[0][1].body as FormData;
  };

  it.each([null, [], ""])(
    "sends an explicit image removal for %j",
    async (imageUrl) => {
      const form = await update({ title: "Entry", imageUrl });
      expect(form.has("imageUrl")).toBe(true);
      expect(form.get("imageUrl")).toBe("");
      expect(form.get("title")).toBe("Entry");
    },
  );

  it.each([
    {},
    { imageUrl: undefined },
    { imageUrl: { src: "https://example.com/old.jpg" } },
  ])(
    "preserves the stored image when no change is requested: %j",
    async (data) => {
      const form = await update(data);
      expect(form.has("imageUrl")).toBe(false);
      expect(form.has("file")).toBe(false);
    },
  );

  it("uploads a replacement without a removal marker", async () => {
    const rawFile = new File(["photo"], "new.jpg", { type: "image/jpeg" });
    const form = await update({ imageUrl: { rawFile } });
    expect(form.get("file")).toBe(rawFile);
    expect(form.has("imageUrl")).toBe(false);
  });

  it("sends an explicit replacement URL", async () => {
    const form = await update({ imageUrl: "https://example.com/new.jpg" });
    expect(form.get("imageUrl")).toBe("https://example.com/new.jpg");
  });

  it("does not add timeline removal fields to other multipart resources", async () => {
    await dataProvider.update("profiles", {
      id: 1,
      data: { imageUrl: null },
      previousData: { id: 1 },
    });
    expect((fetchJson.mock.calls[0][1].body as FormData).has("imageUrl")).toBe(
      false,
    );
  });
});

describe("shared multipart rules", () => {
  it("keeps ordinary JSON updates on the base provider", async () => {
    const params = { id: 1, data: { title: "Team" }, previousData: { id: 1 } };
    baseUpdate.mockResolvedValueOnce({ data: { id: 1, title: "Team" } });
    await dataProvider.update("teams", params);
    expect(baseUpdate).toHaveBeenCalledWith("teams", params);
    expect(fetchJson).not.toHaveBeenCalled();
  });

  const submit = async (
    resource: string,
    data: Record<string, unknown>,
    method: "create" | "update" = "update",
  ) => {
    if (method === "create") {
      await dataProvider.create(resource, { data });
    } else {
      await dataProvider.update(resource, {
        id: 1,
        data,
        previousData: { id: 1 },
      });
    }
    return fetchJson.mock.calls[0][1].body as FormData;
  };

  it.each(["create", "update"] as const)(
    "clears a profile team through %s",
    async (method) => {
      const form = await submit(
        "profiles",
        { team_id: null, rank: "", show: false, position: 0 },
        method,
      );
      expect(form.get("team_id")).toBe("");
      expect(form.get("rank")).toBe("");
      expect(form.get("show")).toBe("false");
      expect(form.get("position")).toBe("0");
    },
  );

  it.each([{}, { team_id: undefined }])(
    "does not clear an omitted or undefined profile team: %j",
    async (data) => {
      const form = await submit("profiles", data);
      expect(form.has("team_id")).toBe(false);
    },
  );

  it("keeps removal rules scoped to the supported resources", async () => {
    const form = await submit("exhibitors", {
      team_id: null,
      imageUrl: null,
      industries: [],
    });
    expect(form.has("team_id")).toBe(false);
    expect(form.has("imageUrl")).toBe(false);
    expect(form.get("industries")).toBe("[]");
  });

  it.each([null, ""])("clears an event image through %j", async (imageUrl) => {
    const form = await submit("events", { imageUrl });
    expect(form.get("imageUrl")).toBe("");
    expect(form.has("file")).toBe(false);
  });

  it.each([{}, { imageUrl: undefined }])(
    "preserves an event image when omitted: %j",
    async (data) => {
      const form = await submit("events", data);
      expect(form.has("imageUrl")).toBe(false);
      expect(form.has("file")).toBe(false);
    },
  );

  it("sends an event replacement file alongside the saved URL", async () => {
    const rawFile = new File(["photo"], "new.jpg", { type: "image/jpeg" });
    const form = await submit("events", {
      imageUrl: "https://example.com/old.jpg",
      file: { rawFile },
    });
    expect(form.get("file")).toBe(rawFile);
    expect(form.get("imageUrl")).toBe("https://example.com/old.jpg");
  });

  it("sends an event replacement URL", async () => {
    const form = await submit("events", {
      imageUrl: "https://example.com/new.jpg",
    });
    expect(form.get("imageUrl")).toBe("https://example.com/new.jpg");
  });

  it("clears a combined blog cover and gallery despite stale saved fields", async () => {
    const form = await submit("blogposts", {
      blogpostImages: [],
      imageUrl: "https://example.com/old.jpg",
      imageUrls: ["https://example.com/old-extra.jpg"],
      headerImages: [{ url: "https://example.com/stale.jpg" }],
    });
    expect(form.getAll("imageUrl")).toEqual([""]);
    expect(form.getAll("headerImages")).toEqual(["[]"]);
    expect(form.has("imageUrls")).toBe(false);
  });

  it("preserves a blog gallery when no image fields are submitted", async () => {
    const form = await submit("blogposts", { title: "Updated title" });
    expect(form.has("imageUrl")).toBe(false);
    expect(form.has("headerImages")).toBe(false);
  });

  it("clears the legacy blog gallery without clearing its cover", async () => {
    const form = await submit("blogposts", { headerImages: [] });
    expect(form.get("headerImages")).toBe("[]");
    expect(form.has("imageUrl")).toBe(false);
  });

  it("retains the ordered blog upload manifest and hides extras without duplicating fields", async () => {
    const rawFile = new File(["photo"], "cover.jpg", { type: "image/jpeg" });
    const form = await submit("blogposts", {
      blogpostImages: [
        { file: { rawFile } },
        { url: "https://example.com/extra.jpg" },
      ],
      imageUrl: "https://example.com/old.jpg",
      showCoverInPost: false,
    });
    expect(form.getAll("file")).toEqual([rawFile]);
    expect(form.getAll("imageUrl")).toEqual([""]);
    expect(form.getAll("headerImages")).toEqual(["[]"]);
  });

  it("rejects unsupported uploads before making a request", async () => {
    const rawFile = new File(["bad"], "bad.pdf", { type: "application/pdf" });
    await expect(
      dataProvider.update("timeline-entries", {
        id: 1,
        data: { imageUrl: { rawFile } },
        previousData: { id: 1 },
      }),
    ).rejects.toThrow("Unsupported file format");
    expect(fetchJson).not.toHaveBeenCalled();
  });
});
