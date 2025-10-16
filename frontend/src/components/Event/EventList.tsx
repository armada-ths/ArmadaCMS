import {
  List,
  Datagrid,
  TextField,
  DateField,
  EditButton,
  ListProps,
  DeleteWithConfirmButton,
} from "react-admin";

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
      <DeleteWithConfirmButton
        confirmTitle="Are you sure?"
        confirmContent="This is PERMANENT"
      />
    </Datagrid>
  </List>
);
