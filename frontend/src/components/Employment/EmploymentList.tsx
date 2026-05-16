import {
  List,
  Datagrid,
  TextField,
  DeleteButton,
  EditButton,
  ListProps,
} from "react-admin";

export const EmploymentList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </List>
);
