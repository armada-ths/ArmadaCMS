import { useEffect, useLayoutEffect, useRef } from "react";
import {
  BooleanInput,
  CheckboxGroupInput,
  SelectInput,
  useSimpleFormIteratorItem,
} from "react-admin";
import { useFormContext, useWatch } from "react-hook-form";
import {
  CHANGE_OWN_PASSWORD_PERM,
  NON_WILDCARD_ACTION_IDS,
  PERMISSION_ACTIONS,
  PermissionGroup,
  useResourceChoices,
} from "./rolePermissionUtils";

/**
 * A single row inside the permissions ArrayInput.
 * Selecting "All actions" automatically ticks every individual action.
 * Unticking any individual action automatically unticks "All actions".
 * Unticking "All actions" clears everything.
 *
 * Uses useLayoutEffect + useRef to correct the value synchronously after each
 * checkbox click, before the browser paints, avoiding any visible flicker.
 */
export const PermissionActionsInput = () => {
  const { index } = useSimpleFormIteratorItem();
  const actionsSource = `permissions.${index}.actions`;
  const { setValue } = useFormContext();
  const currentActions: string[] = useWatch({
    name: actionsSource,
    defaultValue: [],
  });
  const resources = useResourceChoices();
  const prevActionsRef = useRef<string[]>(currentActions);

  useLayoutEffect(() => {
    const prev = prevActionsRef.current;
    const prevHasWildcard = prev.includes("*");
    const nowHasWildcard = currentActions.includes("*");

    if (!prevHasWildcard && nowHasWildcard) {
      // "All actions" just turned on → also select all individual actions
      setValue(actionsSource, ["*", ...NON_WILDCARD_ACTION_IDS], {
        shouldDirty: true,
      });
    } else if (prevHasWildcard && !nowHasWildcard) {
      // "All actions" just turned off → clear everything
      setValue(actionsSource, [], { shouldDirty: true });
    } else if (
      prevHasWildcard &&
      nowHasWildcard &&
      currentActions.length < prev.length
    ) {
      // Wildcard still present but an individual action was unticked → remove wildcard
      setValue(
        actionsSource,
        currentActions.filter((a) => a !== "*"),
        { shouldDirty: true },
      );
    }

    prevActionsRef.current = currentActions;
  }, [currentActions, actionsSource, setValue]);

  return (
    <>
      <SelectInput source="resource" label="Resource" choices={resources} />
      <CheckboxGroupInput
        source="actions"
        label="Actions"
        choices={PERMISSION_ACTIONS}
      />
    </>
  );
};

/** Returns true if the current permissions already cover customusers.changeownpassword. */
const isCoveredByWildcard = (permissions: PermissionGroup[]): boolean =>
  permissions.some(({ resource, actions = [] }) => {
    if (!resource) return false;
    // Global wildcard (*) or customusers.* covers it
    if (
      actions.includes("*") &&
      (resource === "*" || resource === "customusers")
    )
      return true;
    // *.changeownpassword also covers it
    if (
      resource === "*" &&
      actions.includes(CHANGE_OWN_PASSWORD_PERM.split(".")[1])
    )
      return true;
    return false;
  });

/**
 * A toggle for the changeownpassword special permission.
 * Automatically becomes checked and disabled when a wildcard permission already
 * covers customusers.changeownpassword (e.g. "*", "customusers.*").
 */
export const ChangeOwnPasswordInput = () => {
  const { setValue } = useFormContext();
  const permissions: PermissionGroup[] =
    useWatch({ name: "permissions" }) ?? [];
  const covered = isCoveredByWildcard(permissions);

  useEffect(() => {
    if (covered) {
      setValue("changeOwnPassword", true, { shouldDirty: false });
    }
  }, [covered, setValue]);

  return (
    <BooleanInput
      source="changeOwnPassword"
      label="Can change own password"
      disabled={covered}
    />
  );
};
