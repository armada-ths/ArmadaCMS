import { useEffect, useLayoutEffect, useRef } from "react";
import {
  BooleanInput,
  CheckboxGroupInput,
  SelectInput,
  useSimpleFormIteratorItem,
} from "react-admin";
import { useFormContext, useWatch } from "react-hook-form";
import {
  getSpecialPermCoveringWildcards,
  isSpecialPermCoveredByWildcard,
  PERMISSION_ACTIONS,
  PermissionGroup,
  SpecialPermission,
  normalizeActionsSelection,
  useResourceChoices,
} from "./rolePermissionUtils";

/** Set-based equality for action arrays: order-independent comparison. */
const actionsAreEqual = (a: string[], b: string[]) => {
  if (a.length !== b.length) return false;
  const sortedA = [...a].sort();
  const sortedB = [...b].sort();
  return sortedA.every((v, i) => v === sortedB[i]);
};

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

    if (!actionsAreEqual(normalizedActions, currentActions)) {
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
 * A toggle for a single special permission (defined in SPECIAL_PERMISSIONS).
 * Automatically becomes checked and disabled when a wildcard permission already
 * covers it (e.g. "*", "resource.*", "*.action").
 */
export const SpecialPermissionInput = ({
  spec,
}: {
  spec: SpecialPermission;
}) => {
  const { setValue } = useFormContext();
  const permissions: PermissionGroup[] = useWatch({
    name: "permissions",
    defaultValue: [],
  });
  const covered = isSpecialPermCoveredByWildcard(spec, permissions);
  const coveringWildcards = getSpecialPermCoveringWildcards(spec, permissions);

  useEffect(() => {
    if (covered) {
      setValue(spec.formField, true, { shouldDirty: false });
    }
  }, [covered, setValue, spec.formField]);

  return (
    <BooleanInput
      source={spec.formField}
      label={spec.label}
      disabled={covered}
      helperText={
        covered
          ? `Granted by wildcard permission${
              coveringWildcards.length === 1 ? "" : "s"
            }: ${coveringWildcards.join(", ")}`
          : undefined
      }
    />
  );
};
