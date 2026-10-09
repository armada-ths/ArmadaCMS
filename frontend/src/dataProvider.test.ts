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

describe("timeline image updates", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("localStorage", { getItem: () => "test-token" });
    fetchJson.mockResolvedValue({ json: { id: 1 } });
  });

  afterEach(() => vi.unstubAllGlobals());

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
