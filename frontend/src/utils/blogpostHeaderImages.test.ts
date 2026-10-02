import { describe, expect, it } from "vitest";
import { appendBlogpostHeaderImages } from "./blogpostHeaderImages";

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
