import { Chip } from "@mui/material";
import type { FC } from "react";
import {
  List,
  Datagrid,
  TextField,
  DeleteButton,
  EditButton,
  ListProps,
  useRecordContext,
} from "react-admin";

const MAX_VISIBLE_PERMISSIONS = 3;

const PermissionsField: FC<{ label?: string }> = () => {
  const record = useRecordContext();
  const permissions: string[] = record?.permissions ?? [];
  const visible = permissions.slice(0, MAX_VISIBLE_PERMISSIONS);
  const overflow = permissions.length - visible.length;

  return (
    <span style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
      {visible.map((perm) => (
        <Chip key={perm} label={perm} size="small" />
      ))}
      {overflow > 0 && (
        <Chip label={`+${overflow} more`} size="small" variant="outlined" />
      )}
    </span>
  );
};
PermissionsField.displayName = "PermissionsField";

export const RoleList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <PermissionsField label="Permissions" />
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </List>
);
