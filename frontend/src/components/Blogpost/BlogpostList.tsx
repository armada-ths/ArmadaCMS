import {
  List,
  Datagrid,
  TextField,
  DateField,
  EditButton,
  ListProps,
} from "react-admin";
import { PermissionDeleteButton } from "../shared/PermissionDeleteButton";

export const BlogpostList = (props: ListProps) => (
  <List {...props} sort={{ field: "createdAt", order: "DESC" }}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="title" />
      <TextField source="author" />
      <DateField source="createdAt" label="Created" />
      <EditButton />
      <PermissionDeleteButton />
    </Datagrid>
  </List>
);
