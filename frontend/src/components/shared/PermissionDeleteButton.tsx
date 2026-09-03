import {
  DeleteWithConfirmButton,
  usePermissions,
  useResourceContext,
} from "react-admin";

/**
 * A delete button that only renders if the current user has the
 * `<resource>.delete` permission.  Drop-in replacement for
 * `<DeleteWithConfirmButton>` in list Datagrids.
 */
export const PermissionDeleteButton = () => {
  const resource = useResourceContext();
  const { permissions } = usePermissions();
  const perms: string[] = Array.isArray(permissions) ? permissions : [];
  const canDelete = perms.some((p) => p === "*" || p === `${resource}.delete`);

  if (!canDelete) return null;

  return (
    <DeleteWithConfirmButton
      confirmTitle="Are you sure?"
      confirmContent="This is PERMANENT, no backups"
    />
  );
};
