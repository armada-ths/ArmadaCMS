import { useEffect, useLayoutEffect, useRef } from "react";
import {
  BooleanInput,
  CheckboxGroupInput,
  SelectInput,
  useSimpleFormIteratorItem,
} from "react-admin";
import { useFormContext, useWatch } from "react-hook-form";
import {
  getChangeOwnPasswordCoveringWildcards,
  PERMISSION_ACTIONS,
  PermissionGroup,
  isChangeOwnPasswordCoveredByWildcard,
  normalizeActionsSelection,
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
    const previousActions = prevActionsRef.current;
    const normalizedActions = normalizeActionsSelection(
      previousActions,
      currentActions,
    );

    if (
      normalizedActions.length !== currentActions.length ||
      normalizedActions.some(
        (action, actionIndex) => action !== currentActions[actionIndex],
      )
    ) {
      setValue(actionsSource, normalizedActions, { shouldDirty: true });
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

/**
 * A toggle for the changeownpassword special permission.
 * Automatically becomes checked and disabled when a wildcard permission already
 * covers customusers.changeownpassword (e.g. "*", "customusers.*").
 */
export const ChangeOwnPasswordInput = () => {
  const { setValue } = useFormContext();
  const permissions: PermissionGroup[] = useWatch({
    name: "permissions",
    defaultValue: [],
  });
  const covered = isChangeOwnPasswordCoveredByWildcard(permissions);
  const coveringWildcards = getChangeOwnPasswordCoveringWildcards(permissions);

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
      helperText={
        covered
          ? `Granted by wildcard permission${coveringWildcards.length === 1 ? "" : "s"}: ${coveringWildcards.join(", ")}`
          : undefined
      }
    />
  );
};
