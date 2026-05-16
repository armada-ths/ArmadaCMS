import {
  List,
  Datagrid,
  TextField,
  DateField,
  DeleteButton,
  EditButton,
  ListProps,
} from "react-admin";

export const BlogpostList = (props: ListProps) => (
  <List {...props} sort={{ field: "createdAt", order: "DESC" }}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="title" />
      <TextField source="author" />
      <DateField source="createdAt" label="Created" />
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </List>
);
