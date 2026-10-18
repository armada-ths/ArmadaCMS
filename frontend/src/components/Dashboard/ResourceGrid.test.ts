import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { getList, getAccess } = vi.hoisted(() => ({
  getList: vi.fn(),
  getAccess: vi.fn(),
}));
vi.mock("react-admin", () => ({
  useBasename: () => "/admin",
  useGetList: getList,
  useCanAccessResources: getAccess,
}));
import { ResourceGrid } from "./ResourceGrid";
import { RESOURCE_NAMES } from "../../navigation/catalog";

const render = () =>
  renderToStaticMarkup(
    createElement(MemoryRouter, null, createElement(ResourceGrid)),
  );

beforeEach(() => {
  getList.mockReset();
  getList.mockReturnValue({ total: 0, isPending: false, error: null });
  getAccess.mockReset();
  getAccess.mockReturnValue({
    canAccess: Object.fromEntries(RESOURCE_NAMES.map((name) => [name, true])),
    isPending: false,
    error: null,
  });
});

describe("dashboard resource navigation", () => {
  it("shows all 19 resources including empty lists, with Guest photos first", () => {
    const html = render();
    expect(html.match(/href="\/admin\//g)).toHaveLength(19);
    expect(html.indexOf('href="/admin/photoevents"')).toBeLessThan(
      html.indexOf('href="/admin/exhibitors"'),
    );
    expect(html.match(/0 entries/g)).toHaveLength(19);
    expect(html).toContain("Fair &amp; events");
    expect(html).toContain("History timeline");
    expect(getAccess).toHaveBeenCalledWith({
      action: "list",
      resources: RESOURCE_NAMES,
    });
  });

  it("keeps links usable while counts are pending or failing", () => {
    getList.mockImplementation((resource: string) =>
      resource === "photoevents"
        ? { isPending: false, error: new Error("offline") }
        : resource === "events"
          ? { isPending: true }
          : { total: 7, isPending: false },
    );
    const html = render();
    expect(html).toContain("Count unavailable");
    expect(html).toContain('href="/admin/photoevents"');
    expect(html).toContain('href="/admin/events"');
    expect(html).toContain("7 entries");
    expect(html.match(/href="\/admin\//g)).toHaveLength(19);
  });

  it("shows an accessible child and its groups without the parent link", () => {
    getAccess.mockReturnValue({
      canAccess: { industries: true },
      isPending: false,
    });
    const html = render();
    expect(html).toContain('href="/admin/industries"');
    expect(html).not.toContain('href="/admin/exhibitors"');
    expect(html).not.toContain("Administration");
  });

  it("does not fetch counts until permissions load", () => {
    getAccess.mockReturnValue({ isPending: true });
    expect(render()).not.toContain('href="/admin/');
    expect(getList).not.toHaveBeenCalled();
  });

  it("does not expose stale resource links when permission loading fails", () => {
    getAccess.mockReturnValue({
      canAccess: { exhibitors: true },
      isPending: false,
      error: new Error("offline"),
    });
    expect(render()).toContain("Resources unavailable");
    expect(getList).not.toHaveBeenCalled();
  });
});
