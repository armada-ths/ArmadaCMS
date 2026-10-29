import { ResourceList } from "../ResourceList";
import {
  Datagrid,
  TextField,
  DeleteButton,
  EditButton,
  ListProps,
} from "react-admin";

export const EmploymentList = (props: ListProps) => (
  <ResourceList {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </ResourceList>
);
