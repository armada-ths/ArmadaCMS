import {
  List,
  Datagrid,
  TextField,
  DateField,
  EditButton,
  ListProps,
  ReferenceField,
} from "react-admin";
import { PermissionDeleteButton } from "../shared/PermissionDeleteButton";

export const UserList = (props: ListProps) => {
  return (
    <List {...props}>
      <Datagrid>
        <TextField source="username" />
        <TextField source="name" />
        <ReferenceField source="role_id" reference="roles" link={false}>
          <TextField source="name" />
        </ReferenceField>
        <DateField source="created_at" />
        <DateField source="updated_at" />
        <EditButton />
        <PermissionDeleteButton />
      </Datagrid>
    </List>
  );
};
