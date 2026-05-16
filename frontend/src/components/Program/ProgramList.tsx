import {
  List,
  Datagrid,
  TextField,
  DeleteButton,
  EditButton,
  ListProps,
} from "react-admin";

export const ProgramList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </List>
);
