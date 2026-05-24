import { describe, it, expect } from "vitest";
import { hasPerm } from "./permissions";

describe("hasPerm", () => {
  it("returns true for exact match", () => {
    expect(hasPerm(["events.create"], "events.create")).toBe(true);
  });

  it("returns false when permission is absent", () => {
    expect(hasPerm(["events.read"], "events.create")).toBe(false);
  });

  it('returns true for global wildcard "*"', () => {
    expect(hasPerm(["*"], "events.create")).toBe(true);
    expect(hasPerm(["*"], "anything.atall")).toBe(true);
  });

  it('returns true for resource wildcard "resource.*"', () => {
    expect(hasPerm(["events.*"], "events.create")).toBe(true);
    expect(hasPerm(["events.*"], "events.delete")).toBe(true);
  });

  it('does not match a different resource with "resource.*"', () => {
    expect(hasPerm(["exhibitors.*"], "events.create")).toBe(false);
  });

  it('returns true for action wildcard "*.action"', () => {
    expect(hasPerm(["*.create"], "events.create")).toBe(true);
    expect(hasPerm(["*.create"], "exhibitors.create")).toBe(true);
  });

  it('does not match a different action with "*.action"', () => {
    expect(hasPerm(["*.create"], "events.delete")).toBe(false);
  });

  it("returns false for non-dotted required when only dotted wildcards are present", () => {
    expect(hasPerm(["events.*"], "events")).toBe(false);
    expect(hasPerm(["*.create"], "create")).toBe(false);
  });

  it('returns true for non-dotted required only when "*" is present', () => {
    expect(hasPerm(["*"], "admin")).toBe(true);
  });

  it("returns false for an empty permissions list", () => {
    expect(hasPerm([], "events.create")).toBe(false);
  });

  it("returns true when at least one permission in the list matches", () => {
    expect(hasPerm(["events.read", "events.create"], "events.create")).toBe(
      true,
    );
  });

  it("handles malformed permission strings gracefully", () => {
    expect(
      hasPerm(["notadot", ".onlyaction", "resource."], "events.create"),
    ).toBe(false);
  });

  it("does not confuse sub-strings (events.create should not match events.create.extra)", () => {
    expect(hasPerm(["events.create.extra"], "events.create")).toBe(false);
  });
});
