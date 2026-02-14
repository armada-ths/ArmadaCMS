import {
  List,
  Datagrid,
  TextField,
  DateField,
  EditButton,
  ListProps,
} from "react-admin";
import { PermissionDeleteButton } from "../shared/PermissionDeleteButton";

export const EventList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <TextField source="location" />
      <DateField source="eventStart" />
      <DateField source="eventEnd" />
      <TextField source="registrationRequired" />
      <TextField source="show" />
      <EditButton />
      <PermissionDeleteButton />
    </Datagrid>
  </List>
);
