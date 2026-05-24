import { useResourceDefinitions } from "react-admin";

/** One row in the permissions ArrayInput: a resource with one or more selected actions. */
export type PermissionGroup = {
  resource?: string;
  actions?: string[];
};

/** Configuration for a special permission shown as a standalone toggle. */
export type SpecialPermission = {
  /** One or more full permission strings granted/revoked together, e.g. ["customusers.changeownpassword"] */
  perms: string[];
  /** Form field name for the boolean toggle, e.g. "changeOwnPassword" */
  formField: string;
  /** Label shown next to the toggle in the form */
  label: string;
  /**
   * When set, this resource is hidden from the permissions ArrayInput dropdown
   * because all its actions are fully managed by this standalone toggle.
   */
  exclusiveResource?: string;
};

/**
 * All special permissions shown as standalone toggles outside the main
 * permissions ArrayInput. To add a new one, append an entry here — no other
 * changes to this file or the form components are needed.
 */
export const SPECIAL_PERMISSIONS: SpecialPermission[] = [
  {
    perms: ["customusers.changeownpassword"],
    formField: "changeOwnPassword",
    label: "Can change own password",
  },
  {
    perms: ["eventrosync.access"],
    formField: "eventroSyncAccess",
    label: "Can access Eventro sync on the dashboard",
  },
  {
    perms: ["auditlogs.view"],
    formField: "auditLogsAccess",
    label: "Can view audit logs",
    exclusiveResource: "auditlogs",
  },
];

export const PERMISSION_ACTIONS = [
  { id: "*", name: "All actions" },
  { id: "view", name: "View" },
  { id: "create", name: "Create" },
  { id: "edit", name: "Edit" },
  { id: "delete", name: "Delete" },
];

/** Individual (non-wildcard) action IDs — used to expand/collapse the "all actions" shorthand. */
export const NON_WILDCARD_ACTION_IDS = PERMISSION_ACTIONS.filter(
  (a) => a.id !== "*",
).map((a) => a.id);

export const areAllIndividualActionsSelected = (actions: string[]) =>
  NON_WILDCARD_ACTION_IDS.every((action) => actions.includes(action));

export const normalizeActionsSelection = (
  previousActions: string[],
  nextActions: string[],
) => {
  const previousHadWildcard = previousActions.includes("*");
  const nextHasWildcard = nextActions.includes("*");
  const allIndividualsSelected = areAllIndividualActionsSelected(nextActions);

  // Unknown actions are those not in the standard CRUD set (and not "*").
  // They must be preserved through all toggle transitions.
  const unknownNextActions = nextActions.filter(
    (a) => a !== "*" && !NON_WILDCARD_ACTION_IDS.includes(a),
  );

  if (!previousHadWildcard && nextHasWildcard) {
    // "All actions" just checked: expand and preserve unknowns.
    return [...unknownNextActions, "*", ...NON_WILDCARD_ACTION_IDS];
  }

  if (previousHadWildcard && !nextHasWildcard && allIndividualsSelected) {
    // "All actions" just unchecked: collapse CRUD, keep only unknowns.
    return unknownNextActions;
  }

  if (
    previousHadWildcard &&
    nextHasWildcard &&
    nextActions.length < previousActions.length
  ) {
    return nextActions.filter((action) => action !== "*");
  }

  if (!nextHasWildcard && allIndividualsSelected) {
    // All individual CRUD actions manually checked: auto-add wildcard, keep unknowns.
    return [...unknownNextActions, "*", ...NON_WILDCARD_ACTION_IDS];
  }

  return nextActions;
};

/**
 * Returns true if any permission group covers the given special permission
 * via a wildcard ("*", "resource.*", or "*.action").
 */
const isSinglePermCoveredByWildcard = (
  perm: string,
  permissions: PermissionGroup[],
): boolean => {
  const dot = perm.indexOf(".");
  const resource = perm.slice(0, dot);
  const action = perm.slice(dot + 1);

  return permissions.some(({ resource: r, actions = [] }) => {
    if (!r) return false;
    if (actions.includes("*") && (r === "*" || r === resource)) return true;
    if (r === "*" && actions.includes(action)) return true;
    return false;
  });
};

export const isSpecialPermCoveredByWildcard = (
  spec: SpecialPermission,
  permissions: PermissionGroup[],
): boolean =>
  spec.perms.every((p) => isSinglePermCoveredByWildcard(p, permissions));

/**
 * Returns the wildcard permission strings (e.g. "*", "resource.*", "*.action")
 * that cover the given special permission, for use in helper text.
 */
export const getSpecialPermCoveringWildcards = (
  spec: SpecialPermission,
  permissions: PermissionGroup[],
): string[] => {
  const coveringWildcards = new Set<string>();

  for (const perm of spec.perms) {
    const dot = perm.indexOf(".");
    const resource = perm.slice(0, dot);
    const action = perm.slice(dot + 1);

    permissions.forEach(({ resource: r, actions = [] }) => {
      if (!r) return;
      if (actions.includes("*")) {
        if (r === "*") coveringWildcards.add("*");
        if (r === resource) coveringWildcards.add(`${resource}.*`);
      }
      if (r === "*" && !actions.includes("*") && actions.includes(action)) {
        coveringWildcards.add(`*.${action}`);
      }
    });
  }

  return Array.from(coveringWildcards);
};

export const useResourceChoices = () => {
  const resourceDefinitions = useResourceDefinitions();
  const exclusiveResources = new Set(
    SPECIAL_PERMISSIONS.map((sp) => sp.exclusiveResource).filter(Boolean),
  );

  return [
    { id: "*", name: "All resources" },
    ...Object.entries(resourceDefinitions)
      .filter(([resource]) => !exclusiveResources.has(resource))
      .map(([resource, definition]) => ({
        id: resource,
        name: definition.options?.label ?? resource,
      })),
  ];
};

/**
 * Converts stored permission strings from the API into the form's internal shape.
 * Groups by resource into { resource, actions[] } and extracts each special
 * permission into its own boolean form field. When a wildcard action
 * ("resource.*" or "*") is found, all individual action IDs are included so
 * every checkbox appears ticked.
 */
export const normalizeRecord = <T extends { permissions?: string[] }>(
  record: T,
) => {
  const groupMap = new Map<string, string[]>();
  const specialPermSet = new Set(SPECIAL_PERMISSIONS.flatMap((sp) => sp.perms));
  const explicitSpecials = new Set<string>();

  for (const p of record.permissions ?? []) {
    if (specialPermSet.has(p)) {
      explicitSpecials.add(p);
      continue;
    }

    if (p === "*") {
      // Global wildcard — expand so every checkbox is ticked
      groupMap.set("*", ["*", ...NON_WILDCARD_ACTION_IDS]);
      continue;
    }

    const dot = p.indexOf(".");
    if (dot === -1) {
      // Malformed: surface as a resource with no actions so the user can see it.
      groupMap.set(p, groupMap.get(p) ?? []);
      continue;
    }

    const resource = p.slice(0, dot);
    const action = p.slice(dot + 1);

    if (action === "*") {
      // resource.* — expand so every checkbox is ticked for this resource
      groupMap.set(resource, ["*", ...NON_WILDCARD_ACTION_IDS]);
    } else {
      const existing = groupMap.get(resource) ?? [];
      if (!existing.includes(action)) {
        existing.push(action);
      }
      groupMap.set(resource, existing);
    }
  }

  const permissions = Array.from(groupMap.entries()).map(
    ([resource, actions]) => ({ resource, actions }),
  );

  const specialFieldValues: Record<string, boolean> = {};
  for (const spec of SPECIAL_PERMISSIONS) {
    specialFieldValues[spec.formField] =
      spec.perms.some((p) => explicitSpecials.has(p)) ||
      isSpecialPermCoveredByWildcard(spec, permissions);
  }

  return { ...record, permissions, ...specialFieldValues };
};

/**
 * Expands the form's internal shape back to a flat string[] for the API.
 * When "*" is present in a group's actions, emits only the compact wildcard form
 * (e.g. "resource.*") and skips the redundant individual entries.
 * Strips special permission form fields from the payload and folds them back
 * into the permissions array.
 */
export const transformRole = (data: {
  permissions?: PermissionGroup[];
  [key: string]: unknown;
}) => {
  const permissions: string[] = [];
  const permissionGroups = data.permissions ?? [];

  for (const spec of SPECIAL_PERMISSIONS) {
    if (
      data[spec.formField] &&
      !isSpecialPermCoveredByWildcard(spec, permissionGroups)
    ) {
      permissions.push(...spec.perms);
    }
  }

  for (const group of permissionGroups) {
    if (!group.resource || !group.actions?.length) continue;

    if (group.actions.includes("*")) {
      // Compact: wildcard covers all individual actions
      permissions.push(group.resource === "*" ? "*" : `${group.resource}.*`);
    } else {
      for (const action of group.actions) {
        permissions.push(`${group.resource}.${action}`);
      }
    }
  }

  const omitKeys = new Set([
    "permissions",
    ...SPECIAL_PERMISSIONS.map((sp) => sp.formField),
  ]);
  const rest = Object.fromEntries(
    Object.entries(data).filter(([k]) => !omitKeys.has(k)),
  );
  return { ...rest, permissions };
};
