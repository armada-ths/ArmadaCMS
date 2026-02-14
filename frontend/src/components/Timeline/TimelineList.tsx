import { List, Datagrid, TextField, EditButton, ListProps } from "react-admin";
import { PermissionDeleteButton } from "../shared/PermissionDeleteButton";

export const TimelineList = (props: ListProps) => {
  return (
    <List {...props}>
      <Datagrid>
        <TextField source="timeline_title" />
        <TextField source="timeline_date" />
        <EditButton />
        <PermissionDeleteButton />
      </Datagrid>
    </List>
  );
};
