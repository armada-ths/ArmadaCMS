import {
  List,
  Datagrid,
  TextField,
  DateField,
  DeleteButton,
  EditButton,
  ListProps,
  ReferenceField,
} from "react-admin";

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
        <DeleteButton />
      </Datagrid>
    </List>
  );
};
