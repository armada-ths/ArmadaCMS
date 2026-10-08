import { describe, expect, it } from "vitest";
import { toLocalInputValue, toUTCISOString } from "./dateTimeHelpers";

describe("dateTimeHelpers", () => {
  it("converts a browser-local time to an API-compatible UTC value", () => {
    const localValue = "2026-09-26T18:30";
    const parsed = toUTCISOString(localValue);

    expect(parsed).toMatch(/Z$/);
    expect(new Date(parsed!).getTime()).toBe(new Date(localValue).getTime());
    expect(toLocalInputValue(parsed!)).toBe(localValue);
  });

  it("keeps an empty input empty for required-field validation", () => {
    expect(toUTCISOString("")).toBeNull();
  });
});
