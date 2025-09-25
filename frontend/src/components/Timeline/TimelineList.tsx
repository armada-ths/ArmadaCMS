import {
  List,
  Datagrid,
  TextField,
  EditButton,
  ListProps,
  DeleteWithConfirmButton,
} from "react-admin";

export const TimelineList = (props: ListProps) => {
  return (
    <List {...props}>
      <Datagrid>
        <TextField source="timeline_title" />
        <TextField source="timeline_date" />
        <EditButton />
        <DeleteWithConfirmButton
          confirmTitle="Are you sure?"
          confirmContent="This is PERMANENT, no backups"
        />
      </Datagrid>
    </List>
  );
};
