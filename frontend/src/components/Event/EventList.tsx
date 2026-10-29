import { ResourceList } from "../ResourceList";
import {
  Datagrid,
  TextField,
  DateField,
  DeleteButton,
  EditButton,
  ListProps,
} from "react-admin";

export const EventList = (props: ListProps) => (
  <ResourceList {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <TextField source="location" />
      <DateField source="eventStart" />
      <DateField source="eventEnd" />
      <TextField source="registrationRequired" />
      <TextField source="show" />
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </ResourceList>
);
