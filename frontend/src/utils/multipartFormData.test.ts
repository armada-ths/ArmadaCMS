import { describe, expect, it } from "vitest";
import {
  createMultipartFormData,
  isMultipartResource,
} from "./multipartFormData";

describe("lounge JSON mutations", () => {
  it("does not accept uploads for lounge items", () => {
    expect(isMultipartResource("lounge-items")).toBe(false);
    expect(() => createMultipartFormData("lounge-items", {})).toThrow(
      "does not use multipart",
    );
  });
});
