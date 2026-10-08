import { describe, expect, it } from "vitest";
import {
  appendBlogpostHeaderImages,
  getBlogpostImages,
} from "./blogpostHeaderImages";

describe("blog post header images", () => {
  it("preserves the order of retained URLs and multiple new uploads", () => {
    const data = new FormData();
    const first = new File(["photo"], "first.jpg", { type: "image/jpeg" });
    const second = new File(["photo"], "second.png", { type: "image/png" });
    appendBlogpostHeaderImages(data, [
      { file: { rawFile: first }, url: "https://example.com/replaced.jpg" },
      { url: " https://example.com/kept.jpg " },
      { file: { rawFile: second } },
    ]);
    expect(JSON.parse(data.get("headerImages") as string)).toEqual([
      { file: "headerImage0" },
      { url: "https://example.com/kept.jpg" },
      { file: "headerImage2" },
    ]);
    expect(data.get("headerImage0")).toBe(first);
    expect(data.get("headerImage2")).toBe(second);
  });

  it("explicitly sends an empty gallery when all rows are removed", () => {
    const data = new FormData();
    appendBlogpostHeaderImages(data, []);
    expect(data.get("headerImages")).toBe("[]");
  });

  it("validates every upload, including later files", () => {
    expect(() =>
      appendBlogpostHeaderImages(new FormData(), [
        {
          file: { rawFile: new File(["ok"], "ok.jpg", { type: "image/jpeg" }) },
        },
        {
          file: {
            rawFile: new File(["bad"], "bad.pdf", { type: "application/pdf" }),
          },
        },
      ]),
    ).toThrow("Unsupported file format");
  });

  it.each(["", "javascript:alert(1)", "invalid"])(
    "rejects invalid URL %s",
    (url) => {
      expect(() =>
        appendBlogpostHeaderImages(new FormData(), [{ url }]),
      ).toThrow("Header photo 1");
    },
  );
});

describe("combined blog post images", () => {
  it("loads the cover before the additional images", () => {
    expect(
      getBlogpostImages({
        imageUrl: "https://example.com/cover.jpg",
        imageUrls: ["https://example.com/extra.jpg"],
      }),
    ).toEqual([
      { url: "https://example.com/cover.jpg" },
      { url: "https://example.com/extra.jpg" },
    ]);
    expect(getBlogpostImages()).toEqual([]);
  });

  it("promotes a reordered gallery image to cover and retains the old cover", () => {
    const data = new FormData();
    appendBlogpostHeaderImages(
      data,
      [
        { url: " https://example.com/extra.jpg " },
        { url: "https://example.com/cover.jpg" },
      ],
      true,
    );
    expect(data.get("imageUrl")).toBe("https://example.com/extra.jpg");
    expect(JSON.parse(data.get("headerImages") as string)).toEqual([
      { url: "https://example.com/cover.jpg" },
    ]);
  });

  it("uploads the first image as cover and preserves gallery order", () => {
    const data = new FormData();
    const cover = new File(["cover"], "cover.jpg", { type: "image/jpeg" });
    const extra = new File(["extra"], "extra.png", { type: "image/png" });
    appendBlogpostHeaderImages(
      data,
      [
        { file: { rawFile: cover }, url: "https://example.com/old.jpg" },
        { url: "https://example.com/kept.jpg" },
        { file: { rawFile: extra } },
      ],
      true,
    );
    expect(data.get("file")).toBe(cover);
    expect(data.get("imageUrl")).toBe("");
    expect(data.get("headerImage2")).toBe(extra);
    expect(JSON.parse(data.get("headerImages") as string)).toEqual([
      { url: "https://example.com/kept.jpg" },
      { file: "headerImage2" },
    ]);
  });

  it("clears both cover and gallery when all images are removed", () => {
    const data = new FormData();
    appendBlogpostHeaderImages(data, [], true);
    expect(data.get("imageUrl")).toBe("");
    expect(data.get("headerImages")).toBe("[]");
  });
});

describe("hidden header photos", () => {
  it("saves only the cover without losing the extra form images when toggled back on", () => {
    const extra = new File(["extra"], "extra.jpg", { type: "image/jpeg" });
    const images = [
      { url: "https://example.com/cover.jpg" },
      { file: { rawFile: extra } },
      { url: "https://example.com/retained.jpg" },
    ];
    const hidden = new FormData();
    appendBlogpostHeaderImages(hidden, images, true, false);
    expect(hidden.get("imageUrl")).toBe("https://example.com/cover.jpg");
    expect(hidden.get("headerImages")).toBe("[]");
    expect(hidden.has("headerImage1")).toBe(false);
    expect(images).toHaveLength(3);
    expect(images[1].file?.rawFile).toBe(extra);

    const restored = new FormData();
    appendBlogpostHeaderImages(restored, images, true, true);
    expect(restored.get("headerImage1")).toBe(extra);
    expect(JSON.parse(restored.get("headerImages") as string)).toEqual([
      { file: "headerImage1" },
      { url: "https://example.com/retained.jpg" },
    ]);
  });

  it("does not submit or validate an unfinished hidden extra row", () => {
    const data = new FormData();
    appendBlogpostHeaderImages(
      data,
      [{ url: "https://example.com/cover.jpg" }, { url: "" }],
      true,
      false,
    );
    expect(data.get("headerImages")).toBe("[]");
  });
});
