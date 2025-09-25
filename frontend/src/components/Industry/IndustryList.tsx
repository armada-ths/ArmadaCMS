import {
  List,
  Datagrid,
  TextField,
  EditButton,
  ListProps,
  DeleteWithConfirmButton,
} from "react-admin";

export const IndustryList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <EditButton />
      <DeleteWithConfirmButton
        confirmTitle="Are you sure?"
        confirmContent="This is PERMANENT"
      />
    </Datagrid>
  </List>
);
