import {
  List,
  Datagrid,
  TextField,
  DateField,
  EditButton,
  ListProps,
  DeleteWithConfirmButton,
} from "react-admin";

export const UserList = (props: ListProps) => {
  return (
    <List {...props}>
      <Datagrid>
        <TextField source="username" />
        <TextField source="name" />
        <DateField source="created_at" />
        <DateField source="updated_at" />
        <EditButton />
        <DeleteWithConfirmButton
          confirmTitle="Are you sure?"
          confirmContent="This is PERMANENT, no backups"
        />
      </Datagrid>
    </List>
  );
};
