import { describe, expect, it } from "vitest";
import { parsePositiveIntegerId } from "./parsePositiveIntegerId";

describe("positive integer route IDs", () => {
  it.each([
    { value: "1", expected: 1 },
    { value: "0042", expected: 42 },
    { value: "9007199254740991", expected: Number.MAX_SAFE_INTEGER },
  ])("parses $value as $expected", ({ value, expected }) => {
    expect(parsePositiveIntegerId(value)).toBe(expected);
  });

  it.each([
    undefined,
    "",
    "0",
    "-1",
    "+1",
    "1.5",
    "1e2",
    "12abc",
    " 42 ",
    "9007199254740992",
    "Infinity",
  ])("rejects invalid ID %s", (value) => {
    expect(parsePositiveIntegerId(value)).toBeNull();
  });
});
