import { List, Datagrid, TextField, EditButton, ListProps } from "react-admin";
import { PermissionDeleteButton } from "../shared/PermissionDeleteButton";

export const IndustryList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <EditButton />
      <PermissionDeleteButton />
    </Datagrid>
  </List>
);
