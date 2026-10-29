import { ResourceList } from "../ResourceList";
import {
  Datagrid,
  TextField,
  BooleanField,
  DateField,
  DeleteButton,
  EditButton,
  ListProps,
} from "react-admin";

export const BlogpostList = (props: ListProps) => (
  <ResourceList {...props} sort={{ field: "createdAt", order: "DESC" }}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="title" />
      <TextField source="author" />
      <BooleanField source="published" />
      <DateField source="createdAt" label="Created" />
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </ResourceList>
);
