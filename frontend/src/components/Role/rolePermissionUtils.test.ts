import { describe, expect, it } from "vitest";
import {
  RESOURCE_ACTION_OVERRIDES,
  getPermissionActionChoices,
  normalizeRecord,
  SPECIAL_PERMISSIONS,
  transformRole,
} from "./rolePermissionUtils";

const photoPermissions = [
  "photoevents.view",
  "photoevents.create",
  "photoevents.edit",
  "photoevents.delete",
  "eventphotos.view",
  "eventphotos.edit",
  "photoexports.view",
  "photoexports.create",
];

describe("guest photo role permissions", () => {
  it("keeps photo permissions in the regular resource selector", () => {
    expect(
      SPECIAL_PERMISSIONS.flatMap((spec) => spec.perms).filter((permission) =>
        photoPermissions.includes(permission),
      ),
    ).toEqual([]);
    expect(RESOURCE_ACTION_OVERRIDES).toEqual([
      { id: "auditlogs", name: "Audit logs", actions: ["view"] },
      {
        id: "eventphotos",
        name: "Guest photo moderation",
        actions: ["view", "edit"],
      },
      {
        id: "photoexports",
        name: "Photo exports",
        actions: ["view", "create"],
      },
    ]);
  });

  it("offers only supported actions for API-only resources", () => {
    expect(getPermissionActionChoices("auditlogs").map(({ id }) => id)).toEqual(
      ["*", "view"],
    );
    expect(
      getPermissionActionChoices("eventphotos").map(({ id }) => id),
    ).toEqual(["*", "view", "edit"]);
    expect(
      getPermissionActionChoices("photoexports").map(({ id }) => id),
    ).toEqual(["*", "view", "create"]);
    expect(
      getPermissionActionChoices("photoevents").map(({ id }) => id),
    ).toEqual(["*", "view", "create", "edit", "delete"]);
    expect(
      getPermissionActionChoices("eventphotos", ["delete"]).map(({ id }) => id),
    ).toContain("delete");
  });

  it("edits audit log view as a regular permission while retaining genuine special actions", () => {
    const record = normalizeRecord({
      permissions: [
        "auditlogs.view",
        "customusers.changeownpassword",
        "eventrosync.access",
      ],
    });
    expect(record.permissions).toEqual([
      { resource: "auditlogs", actions: ["view"] },
    ]);
    expect(record.changeOwnPassword).toBe(true);
    expect(record.eventroSyncAccess).toBe(true);
    expect(transformRole(record).permissions).toEqual([
      "customusers.changeownpassword",
      "eventrosync.access",
      "auditlogs.view",
    ]);
  });

  it("loads and saves existing photo permissions without granting others", () => {
    const record = normalizeRecord({
      name: "Photo moderator",
      permissions: ["photoevents.view", "eventphotos.view", "eventphotos.edit"],
    });

    expect(record.permissions).toEqual([
      { resource: "photoevents", actions: ["view"] },
      { resource: "eventphotos", actions: ["view", "edit"] },
    ]);
    expect(transformRole(record)).toEqual({
      name: "Photo moderator",
      permissions: ["photoevents.view", "eventphotos.view", "eventphotos.edit"],
    });
  });

  it("keeps a resource wildcard compact without duplicating photo permissions", () => {
    const record = normalizeRecord({ permissions: ["photoevents.*"] });
    expect(record.permissions).toEqual([
      {
        resource: "photoevents",
        actions: ["*", "view", "create", "edit", "delete"],
      },
    ]);
    expect(transformRole(record).permissions).toEqual(["photoevents.*"]);
  });
});
