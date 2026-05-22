import { useResourceDefinitions } from "react-admin";

/** One row in the permissions ArrayInput: a resource with one or more selected actions. */
export type PermissionGroup = {
  resource?: string;
  actions?: string[];
};

/** The special-cased permission that lives outside the main ArrayInput. */
export const CHANGE_OWN_PASSWORD_PERM = "customusers.changeownpassword";
export const CHANGE_OWN_PASSWORD_ACTION =
  CHANGE_OWN_PASSWORD_PERM.split(".")[1] ?? "";

export const PERMISSION_ACTIONS = [
  { id: "*", name: "All actions" },
  { id: "list", name: "List" },
  { id: "show", name: "Show" },
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

  if (!previousHadWildcard && nextHasWildcard) {
    return ["*", ...NON_WILDCARD_ACTION_IDS];
  }

  if (previousHadWildcard && !nextHasWildcard && allIndividualsSelected) {
    return [];
  }

  if (
    previousHadWildcard &&
    nextHasWildcard &&
    nextActions.length < previousActions.length
  ) {
    return nextActions.filter((action) => action !== "*");
  }

  if (!nextHasWildcard && allIndividualsSelected) {
    return ["*", ...NON_WILDCARD_ACTION_IDS];
  }

  return nextActions;
};

export const isChangeOwnPasswordCoveredByWildcard = (
  permissions: PermissionGroup[],
) =>
  permissions.some(({ resource, actions = [] }) => {
    if (!resource) {
      return false;
    }

    if (
      actions.includes("*") &&
      (resource === "*" || resource === "customusers")
    ) {
      return true;
    }

    if (resource === "*" && actions.includes(CHANGE_OWN_PASSWORD_ACTION)) {
      return true;
    }

    return false;
  });

export const getChangeOwnPasswordCoveringWildcards = (
  permissions: PermissionGroup[],
) => {
  const coveringWildcards = new Set<string>();

  permissions.forEach(({ resource, actions = [] }) => {
    if (!resource) {
      return;
    }

    if (actions.includes("*")) {
      if (resource === "*") {
        coveringWildcards.add("*");
      }

      if (resource === "customusers") {
        coveringWildcards.add("customusers.*");
      }
    }

    if (resource === "*" && actions.includes(CHANGE_OWN_PASSWORD_ACTION)) {
      coveringWildcards.add(`*.${CHANGE_OWN_PASSWORD_ACTION}`);
    }
  });

  return Array.from(coveringWildcards);
};

export const useResourceChoices = () => {
  const resourceDefinitions = useResourceDefinitions();

  return [
    { id: "*", name: "All resources" },
    ...Object.entries(resourceDefinitions).map(([resource, definition]) => ({
      id: resource,
      name: definition.options?.label ?? resource,
    })),
  ];
};

/**
 * Converts stored permission strings from the API into the form's internal shape.
 * Groups by resource into { resource, actions[] } and extracts the standalone
 * changeownpassword flag. When a wildcard action ("resource.*" or "*") is found,
 * all individual action IDs are included so every checkbox appears ticked.
 */
export const normalizeRecord = <T extends { permissions?: string[] }>(
  record: T,
) => {
  const groupMap = new Map<string, string[]>();
  let hasExplicitChangeOwnPassword = false;

  for (const p of record.permissions ?? []) {
    if (p === CHANGE_OWN_PASSWORD_PERM) {
      hasExplicitChangeOwnPassword = true;
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
      existing.push(action);
      groupMap.set(resource, existing);
    }
  }

  const permissions = Array.from(groupMap.entries()).map(
    ([resource, actions]) => ({
      resource,
      actions,
    }),
  );

  return {
    ...record,
    permissions,
    changeOwnPassword:
      hasExplicitChangeOwnPassword ||
      isChangeOwnPasswordCoveredByWildcard(permissions),
  };
};

/**
 * Expands the form's internal shape back to a flat string[] for the API.
 * When "*" is present in a group's actions, emits only the compact wildcard form
 * (e.g. "resource.*") and skips the redundant individual entries.
 * Strips changeOwnPassword from the payload and folds it into permissions.
 */
export const transformRole = (data: {
  permissions?: PermissionGroup[];
  changeOwnPassword?: boolean;
  [key: string]: unknown;
}) => {
  const permissions: string[] = [];
  const permissionGroups = data.permissions ?? [];
  const changeOwnPasswordCoveredByWildcard =
    isChangeOwnPasswordCoveredByWildcard(permissionGroups);

  if (data.changeOwnPassword && !changeOwnPasswordCoveredByWildcard) {
    permissions.push(CHANGE_OWN_PASSWORD_PERM);
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

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { changeOwnPassword: _omit, permissions: _perms, ...rest } = data;
  return { ...rest, permissions };
};
