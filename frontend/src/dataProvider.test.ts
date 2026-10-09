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

import { dataProvider } from "./dataProvider";

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
    const form = await submit("events", {
      team_id: null,
      imageUrl: null,
      industries: [],
    });
    expect(form.has("team_id")).toBe(false);
    expect(form.has("imageUrl")).toBe(false);
    expect(form.get("industries")).toBe("[]");
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
