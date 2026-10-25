import appSource from "../App.tsx?raw";
import { afterEach, describe, expect, it, vi } from "vitest";
import { authProvider } from "../context/authProvider";
import {
  filterNavigation,
  getActiveGroupIds,
  getResourceLinks,
  isNavigationActive,
  NAVIGATION,
  RESOURCE_NAMES,
} from "./catalog";

afterEach(() => vi.unstubAllGlobals());

const accessibleNavigation = async (permissions: string[]) => {
  const payload = btoa(JSON.stringify({ permissions }));
  vi.stubGlobal("localStorage", {
    getItem: () => `header.${payload}.signature`,
  });
  const access = Object.fromEntries(
    await Promise.all(
      RESOURCE_NAMES.map(async (resource) => [
        resource,
        await authProvider.canAccess!({ action: "list", resource }),
      ]),
    ),
  );
  return filterNavigation(NAVIGATION, access);
};

describe("navigation catalog", () => {
  it("covers all registered resources exactly once", () => {
    const registered = [
      ...appSource.matchAll(/<Resource\s+name="([^"]+)"/g),
    ].map((match) => match[1]);
    expect(registered).toHaveLength(19);
    expect(RESOURCE_NAMES).toHaveLength(19);
    expect(new Set(RESOURCE_NAMES).size).toBe(19);
    expect([...RESOURCE_NAMES].sort()).toEqual(registered.sort());
  });

  it("keeps Guest photos on the top level after Dashboard", () => {
    expect(NAVIGATION.slice(0, 2).map((node) => node.id)).toEqual([
      "dashboard",
      "photoevents",
    ]);
    expect(NAVIGATION.slice(2).map((node) => node.label)).toEqual([
      "Fair & events",
      "Website content",
      "Organisation",
      "Recruitment",
      "Administration",
    ]);
  });

  it.each(["*", "*.view"])(
    "shows every resource for %s",
    async (permission) => {
      expect(
        getResourceLinks(await accessibleNavigation([permission])),
      ).toHaveLength(19);
    },
  );

  it("preserves child access without granting access to its parent", async () => {
    const nodes = await accessibleNavigation(["industries.view"]);
    expect(nodes.map((node) => node.id)).toEqual(["dashboard", "fair"]);
    expect(getResourceLinks(nodes).map((node) => node.resource)).toEqual([
      "industries",
    ]);
    expect(getActiveGroupIds(nodes, "/admin/industries/3", "/admin")).toEqual([
      "fair",
      "exhibitor-settings",
    ]);
  });

  it("supports resource wildcards and hides empty groups", async () => {
    const nodes = await accessibleNavigation(["photoevents.*", "teams.view"]);
    expect(nodes.map((node) => node.id)).toEqual([
      "dashboard",
      "photoevents",
      "organisation",
    ]);
    expect(getResourceLinks(nodes).map((node) => node.resource)).toEqual([
      "photoevents",
      "teams",
    ]);
  });

  it("does not show list links for create-only permissions", async () => {
    expect(await accessibleNavigation(["industries.create"])).toEqual([
      NAVIGATION[0],
    ]);
  });

  it.each([
    ["/admin/exhibitors", ["fair", "exhibitor-settings"]],
    ["/admin/exhibitors/create", ["fair", "exhibitor-settings"]],
    ["/admin/programs/17", ["fair", "exhibitor-settings"]],
    ["/admin/timeline-entries/4", ["website", "history"]],
    ["/admin/photoevents/2/photos", []],
    ["/admin/", []],
  ])("opens the ancestors for %s", (path, expected) => {
    expect(getActiveGroupIds(NAVIGATION, path, "/admin")).toEqual(expected);
  });

  it("marks Guest photos active for moderation without prefix collisions", () => {
    const photos = NAVIGATION[1];
    expect(
      isNavigationActive(photos, "/admin/photoevents/2/photos", "/admin"),
    ).toBe(true);
    expect(
      isNavigationActive(photos, "/admin/photoevents-other", "/admin"),
    ).toBe(false);
    expect(isNavigationActive(NAVIGATION[0], "/admin/events", "/admin")).toBe(
      false,
    );
  });
});
