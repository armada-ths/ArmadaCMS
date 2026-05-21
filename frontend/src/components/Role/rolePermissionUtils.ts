import { useResourceDefinitions } from "react-admin";

/** One row in the permissions ArrayInput: a resource with one or more selected actions. */
export type PermissionGroup = {
  resource?: string;
  actions?: string[];
};

/** The special-cased permission that lives outside the main ArrayInput. */
const CHANGE_OWN_PASSWORD_PERM = "customusers.changeownpassword";

export const PERMISSION_ACTIONS = [
  { id: "*", name: "All actions (*)" },
  { id: "list", name: "List" },
  { id: "show", name: "Show" },
  { id: "create", name: "Create" },
  { id: "edit", name: "Edit" },
  { id: "delete", name: "Delete" },
];

export const useResourceChoices = () => {
  const resourceDefinitions = useResourceDefinitions();

  return [
    { id: "*", name: "All resources (*)" },
    ...Object.entries(resourceDefinitions).map(([resource, definition]) => ({
      id: resource,
      name: definition.options?.label ?? resource,
    })),
  ];
};

/**
 * Converts stored permission strings from the API into the form's internal shape:
 * groups by resource into { resource, actions[] } and extracts the standalone
 * changeownpassword flag.
 */
export const normalizeRecord = <T extends { permissions?: string[] }>(
  record: T,
) => {
  const groupMap = new Map<string, string[]>();
  let changeOwnPassword = false;

  for (const p of record.permissions ?? []) {
    if (p === CHANGE_OWN_PASSWORD_PERM) {
      changeOwnPassword = true;
      continue;
    }

    if (p === "*") {
      groupMap.set("*", ["*"]);
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
    const existing = groupMap.get(resource) ?? [];
    existing.push(action);
    groupMap.set(resource, existing);
  }

  return {
    ...record,
    permissions: Array.from(groupMap.entries()).map(([resource, actions]) => ({
      resource,
      actions,
    })),
    changeOwnPassword,
  };
};

/**
 * Expands the form's internal shape back to a flat string[] for the API.
 * Strips the changeOwnPassword field and folds it into permissions.
 */
export const transformRole = (data: {
  permissions?: PermissionGroup[];
  changeOwnPassword?: boolean;
  [key: string]: unknown;
}) => {
  const permissions: string[] = [];

  if (data.changeOwnPassword) {
    permissions.push(CHANGE_OWN_PASSWORD_PERM);
  }

  for (const group of data.permissions ?? []) {
    if (!group.resource || !group.actions?.length) continue;
    for (const action of group.actions) {
      if (group.resource === "*" && action === "*") {
        permissions.push("*");
      } else {
        permissions.push(`${group.resource}.${action}`);
      }
    }
  }

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { changeOwnPassword: _omit, permissions: _perms, ...rest } = data;
  return { ...rest, permissions };
};
