import {
  List,
  Datagrid,
  TextField,
  NumberField,
  DeleteButton,
  EditButton,
  ListProps,
} from "react-admin";

export const TimelineEntryList = (props: ListProps) => {
  return (
    <List {...props}>
      <Datagrid>
        <TextField source="title" />
        <TextField source="era" />
        <NumberField source="sortOrder" label="Sort order" />
        <EditButton />
        <DeleteButton />
      </Datagrid>
    </List>
  );
};
