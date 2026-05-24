import { Chip } from "@mui/material";
import type { FC } from "react";
import {
  List,
  Datagrid,
  TextField,
  DateField,
  DeleteButton,
  EditButton,
  ListProps,
  useRecordContext,
} from "react-admin";

const MAX_VISIBLE_ROLES = 3;

interface Role {
  id: number;
  name: string;
}

const RolesField: FC<{ label?: string }> = () => {
  const record = useRecordContext();
  const roles: Role[] = record?.roles ?? [];
  const visible = roles.slice(0, MAX_VISIBLE_ROLES);
  const overflow = roles.length - visible.length;

  return (
    <span style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
      {visible.map((role) => (
        <Chip key={role.id} label={role.name} size="small" />
      ))}
      {overflow > 0 && (
        <Chip label={`+${overflow} more`} size="small" variant="outlined" />
      )}
    </span>
  );
};
RolesField.displayName = "RolesField";

export const UserList = (props: ListProps) => {
  return (
    <List {...props}>
      <Datagrid>
        <TextField source="username" />
        <TextField source="name" />
        <RolesField label="Roles" />
        <DateField source="created_at" />
        <DateField source="updated_at" />
        <EditButton />
        <DeleteButton />
      </Datagrid>
    </List>
  );
};
